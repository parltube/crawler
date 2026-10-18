"""
Общая часть для проб лимитов YouTube: сессии, парсинг, рамп, итог.
Трафик идёт как есть (TUN/VPN настраивается на уровне ОС).

Каждая проба (watch.py, browse.py, player.py, comments.py) определяет
функцию fn(target) -> (status, bytes, ok, note, items, fields)
и вызывает run(...). Сырые результаты пишутся в data/<ep>_<label>.jsonl.
"""
import asyncio, json, time, argparse, re, random, threading
from collections import Counter, defaultdict, deque
from pathlib import Path
from curl_cffi import requests as cr

UA_MWEB = ("Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 "
           "(KHTML, like Gecko) Chrome/136.0.0.0 Mobile Safari/537.36")
UA_WEB = ("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
          "(KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

DATA = Path(__file__).parent / "data"

SEED = ["dQw4w9WgXcQ","9bZkp7q19f0","kJQP7kiw5Fk","JGwWNGJdvx8","OPf0YbXqDm0",
        "RgKAFK5djSk","fJ9rUzIMcZQ","CevxZvSJLk8","hT_nvWreIhg","YQHsXMglC9A",
        "09R8_2nJtjg","2Vv-BfVoq4g","YbJOTdZBX1g","1G4isv_Fylg","e-ORhEE9VVg"]

CFG = {}
_tl = threading.local()


def S():
    """Сессия на поток, прогретая заходом на m.youtube.com."""
    if not hasattr(_tl, "s"):
        s = cr.Session()
        try:
            s.get("https://m.youtube.com/", impersonate=CFG["imp"],
                  headers={"User-Agent": UA_MWEB}, timeout=25)
        except Exception:
            pass
        _tl.s = s
    return _tl.s


RE_PR = re.compile(rb"ytInitialPlayerResponse\s*=\s*\{")


def blob(buf):
    """Вырезает JSON ytInitialPlayerResponse из HTML страницы."""
    for m in RE_PR.finditer(buf):
        j = buf.find(b"{", m.end() - 1)
        if j < 0:
            continue
        d = 0; instr = False; esc = False; k = j; n = len(buf)
        while k < n:
            c = buf[k]
            if instr:
                if esc: esc = False
                elif c == 0x5c: esc = True
                elif c == 0x22: instr = False
            else:
                if c == 0x22: instr = True
                elif c == 0x7b: d += 1
                elif c == 0x7d:
                    d -= 1
                    if d == 0: return buf[j:k+1]
            k += 1
    return None


def find(o, name, out=None):
    """Все значения по ключу name на любой глубине."""
    out = [] if out is None else out
    if isinstance(o, dict):
        for k, v in o.items():
            if k == name: out.append(v)
            else: find(v, name, out)
    elif isinstance(o, list):
        for v in o: find(v, name, out)
    return out


FIELDS = ["status_OK","videoId","title","channelId","viewCount","lengthSeconds",
          "shortDesc","keywords","mf_likeCount","mf_publishDate","mf_category",
          "mf_countries","mf_desc","captions","streamingData"]
EMPTY = {f: False for f in FIELDS}


def fmap(d):
    """Какие поля playerResponse пришли заполненными."""
    ps = d.get("playabilityStatus") or {}
    vd = d.get("videoDetails") or {}
    mf = (d.get("microformat") or {}).get("playerMicroformatRenderer") or {}
    st = d.get("streamingData") or {}
    cap = ((d.get("captions") or {}).get("playerCaptionsTracklistRenderer") or {})
    return {
        "status_OK": ps.get("status") == "OK",
        "videoId": bool(vd.get("videoId")),
        "title": len(vd.get("title") or "") > 2,
        "channelId": bool(vd.get("channelId")),
        "viewCount": str(vd.get("viewCount","")).isdigit() and int(vd.get("viewCount",0)) > 0,
        "lengthSeconds": str(vd.get("lengthSeconds","")).isdigit(),
        "shortDesc": bool(vd.get("shortDescription")),
        "keywords": bool(vd.get("keywords")),
        "mf_likeCount": str(mf.get("likeCount","")).isdigit(),
        "mf_publishDate": bool(mf.get("publishDate")),
        "mf_category": bool(mf.get("category")),
        "mf_countries": bool(mf.get("availableCountries")),
        "mf_desc": bool(mf.get("description")),
        "captions": bool(cap.get("captionTracks")),
        "streamingData": bool(st.get("formats") or st.get("adaptiveFormats")),
    }


# ---------------------------------------------------------------- innertube

def post(path, body, cname, ua, ref):
    h = {"Content-Type": "application/json", "User-Agent": ua,
         "X-YouTube-Client-Name": cname,
         "X-YouTube-Client-Version": CFG["ver"],
         "Origin": "https://www.youtube.com", "Referer": ref}
    if CFG.get("vis"):
        h["X-Goog-Visitor-Id"] = CFG["vis"]
    return S().post(f"https://www.youtube.com/youtubei/v1/{path}"
                    f"?key={CFG['key']}&prettyPrint=false",
                    json=body, headers=h, impersonate=CFG["imp"], timeout=30)


def ctx(name):
    c = {"clientName": name, "clientVersion": CFG["ver"], "hl": "en", "gl": "US"}
    if CFG.get("vis"): c["visitorData"] = CFG["vis"]
    return {"client": c}


def bootstrap():
    """Достаёт ключ/версию/visitorData innertube со страницы видео."""
    s = cr.Session()
    s.get("https://m.youtube.com/", impersonate=CFG["imp"],
          headers={"User-Agent": UA_MWEB}, timeout=30)
    h = s.get("https://m.youtube.com/watch?v=dQw4w9WgXcQ",
              impersonate=CFG["imp"], headers={"User-Agent": UA_MWEB},
              timeout=30).text
    CFG["key"] = (re.search(r'"INNERTUBE_API_KEY":"([^"]+)"', h) or [None,""])[1]
    CFG["ver"] = (re.search(r'"INNERTUBE_CLIENT_VERSION":"([^"]+)"', h) or [None,""])[1]
    CFG["vis"] = (re.search(r'"visitorData":"([^"]+)"', h) or [None,""])[1]
    print(f"  ver={CFG['ver']} key={'ok' if CFG['key'] else 'FAIL'} "
          f"vis={'ok' if CFG['vis'] else 'no'}")
    return bool(CFG["key"])


# ---------------------------------------------------------------- проверки

def header(title):
    print("\n" + "="*72)
    print(title)
    print("="*72)


def sanity():
    header("SANITY  (сверь IP с тем, что показывает браузер на Windows!)")
    s = cr.Session()
    for url, label in (("https://api.ipify.org?format=json", "ipify"),
                       ("https://tls.browserleaks.com/json", "browserleaks")):
        try:
            j = s.get(url, impersonate=CFG["imp"], timeout=25).json()
            if label == "ipify":
                print(f"  IP видимый : {j.get('ip')}")
            else:
                print(f"  IP (tls)   : {j.get('ip')}")
                print(f"  JA3        : {j.get('ja3_hash')}")
                print(f"  JA4        : {j.get('ja4')}")
        except Exception as e:
            print(f"  {label}: FAIL {type(e).__name__}")
    try:
        r = s.get("https://m.youtube.com/watch?v=dQw4w9WgXcQ",
                  headers={"User-Agent": UA_MWEB},
                  impersonate=CFG["imp"], timeout=30)
        b = blob(r.content)
        if not b:
            print(f"  YouTube    : БЛОК, нет playerResponse ({len(r.content)}B)")
            return False
        d = json.loads(b)
        vd = d.get("videoDetails", {})
        mf = d.get("microformat", {}).get("playerMicroformatRenderer", {})
        print(f"  YouTube    : OK  {len(r.content)}B  "
              f"status={d.get('playabilityStatus',{}).get('status')}")
        print(f"  views={vd.get('viewCount')} likes={mf.get('likeCount')}")
        return True
    except Exception as e:
        print(f"  YouTube    : EXC {type(e).__name__} {e}")
        return False


# ---------------------------------------------------------------- рамп

class St:
    def __init__(self):
        self.n = self.ok = self.bytes = self.items = 0
        self.notes = Counter(); self.lat = deque(maxlen=50000)
        self.win = deque(maxlen=150); self.fields = defaultdict(int)
        self.first_fail_n = None

    def add(self, nb, ok, note, ni, fm, dt):
        self.n += 1; self.notes[note] += 1
        self.bytes += nb; self.items += ni; self.lat.append(dt)
        self.win.append(1 if ok else 0)
        for k, v in fm.items():
            if v: self.fields[k] += 1
        if ok: self.ok += 1
        elif self.first_fail_n is None: self.first_fail_n = self.n

    @property
    def wr(self):
        return sum(self.win)/len(self.win) if self.win else 1.0

    def p(self, q):
        if not self.lat: return 0
        s = sorted(self.lat); return s[min(len(s)-1, int(len(s)*q))]


async def ramp(ep, fn, targets, steps, step_sec, conc, break_at, fh,
               items=None, fields=False):
    """items: подпись для штук/с (напр. "vid"), fields: печатать неполные поля."""
    header(f"ЭТАП {ep.upper()}: рамп {steps} по {step_sec}с")
    loop = asyncio.get_running_loop()
    results = []
    for rps in steps:
        st = St(); sem = asyncio.Semaphore(conc); t0 = time.perf_counter()

        async def one(tgt):
            async with sem:
                s0 = time.perf_counter()
                try:
                    _, nb, ok, note, ni, fm = await loop.run_in_executor(None, fn, tgt)
                except Exception as e:
                    nb, ok, note, ni, fm = 0, False, f"EXC:{type(e).__name__}", 0, {}
                dt = time.perf_counter() - s0
                st.add(nb, ok, note, ni, fm, dt)
                fh.write(json.dumps({"ep": ep, "rps": rps, "ok": ok,
                                     "note": note, "bytes": nb, "items": ni,
                                     "ms": round(dt*1000)}) + "\n")

        deadline = t0 + step_sec; iv = 1.0/rps
        nxt = time.perf_counter(); last = nxt; tasks = []
        while time.perf_counter() < deadline:
            tasks.append(asyncio.create_task(one(random.choice(targets))))
            nxt += iv
            slp = nxt - time.perf_counter()
            if slp > 0: await asyncio.sleep(slp)
            else: nxt = time.perf_counter()
            now = time.perf_counter()
            if now - last >= 20:
                ex = f" {items}/s={st.items/(now-t0):5.1f}" if items else ""
                print(f"    [{now-t0:4.0f}s] n={st.n} ok={st.ok} "
                      f"win={st.wr*100:5.1f}%{ex} p50={st.p(.5)*1000:5.0f}ms "
                      f"{dict(st.notes.most_common(2))}")
                last = now
            tasks = [t for t in tasks if not t.done()]
        if tasks:
            await asyncio.gather(*tasks, return_exceptions=True)
        el = time.perf_counter() - t0
        line = (f"  @{rps:5.1f} rps: n={st.n:5} ok={st.ok:5} "
                f"({100*st.ok/max(1,st.n):5.1f}%) eff={st.ok/el:6.2f} "
                f"avg={st.bytes/max(1,st.n)/1024:7.1f}KB")
        if items:
            line += (f" {items}/s={st.items/el:6.1f}"
                     f" B/{items}={st.bytes/max(1,st.items):.0f}")
        print(line)
        print(f"         notes={dict(st.notes)}")
        if st.first_fail_n:
            print(f"         первый отказ на #{st.first_fail_n}")
        if fields and st.n:
            weak = [(f, 100*st.fields[f]/st.n) for f in FIELDS
                    if st.fields[f]/st.n < 0.97]
            if weak:
                print("         неполные: " +
                      ", ".join(f"{f}={p:.0f}%" for f, p in weak))
        results.append((rps, st))
        if st.wr < break_at:
            print(f"  *** СЛОМ на {rps} rps ***")
            break
    return results


def summary(ep, res, break_at, label, items=None):
    header(f"ИТОГ [{label}]")
    good = [r for r in res if r[1].wr >= break_at]
    if good:
        rps, st = max(good, key=lambda x: x[0])
        extra = f", {st.items/max(1,st.n)*rps:.0f} {items}/с" if items else ""
        print(f"  {ep:8} держит до {rps} rps{extra}")
    else:
        n = res[0][1].notes.most_common(1)
        print(f"  {ep:8} НЕ РАБОТАЕТ  {n[0][0] if n else ''}")


# ---------------------------------------------------------------- cli

def parser():
    p = argparse.ArgumentParser()
    p.add_argument("--label", default="run")
    p.add_argument("--ids", help="файл с id видео, по одному на строку")
    p.add_argument("--conc", type=int, default=48)
    p.add_argument("--break-at", type=float, default=0.9)
    p.add_argument("--imp", default="chrome131")
    p.add_argument("--quick", action="store_true", help="короткий рамп ~3 мин")
    return p


def setup(a):
    """Конфиг + sanity + bootstrap. False — дальше смысла нет."""
    CFG["imp"] = a.imp
    CFG["ids"] = [l.strip() for l in open(a.ids) if l.strip()] if a.ids else SEED
    if not sanity():
        print("\nIP не проходит базовую проверку — дальше смысла нет.")
        return False
    if not bootstrap():
        print("bootstrap FAILED")
        return False
    return True


def run(a, ep, fn, targets=None, items=None, fields=False):
    """Полный прогон одной пробы: setup -> рамп -> итог."""
    if not setup(a):
        return
    steps = [10,50,80] if a.quick else [10,20,40,80,40,80]
    ssec = 60 if a.quick else 180
    DATA.mkdir(exist_ok=True)
    with open(DATA / f"{ep}_{a.label}.jsonl", "w") as fh:
        res = asyncio.run(ramp(ep, fn, targets or CFG["ids"], steps, ssec,
                               a.conc, a.break_at, fh, items, fields))
    summary(ep, res, a.break_at, a.label, items)


if __name__ == "__main__":
    a = parser().parse_args()
    CFG["imp"] = a.imp
    sanity()
