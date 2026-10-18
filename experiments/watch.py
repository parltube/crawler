"""
Проба m.youtube.com/watch (HTML + ytInitialPlayerResponse).

    python watch.py --label isp-tun --quick
    python watch.py --label isp-tun --integrity   # проверка на подделку/кэш
"""
import json
import concurrent.futures
import utils as U


def do_watch(vid):
    r = U.S().get(f"https://m.youtube.com/watch?v={vid}",
                  headers={"User-Agent": U.UA_MWEB,
                           "Accept-Language": "en-US,en;q=0.9"},
                  impersonate=U.CFG["imp"], timeout=30)
    body = r.content
    if b"consent" in body[:5000].lower():
        return r.status_code, len(body), False, "consent", 0, U.EMPTY
    b = U.blob(body)
    if not b:
        return r.status_code, len(body), False, "no_blob", 0, U.EMPTY
    try:
        d = json.loads(b)
    except Exception:
        return r.status_code, len(body), False, "bad_json", 0, U.EMPTY
    fm = U.fmap(d)
    st = (d.get("playabilityStatus") or {}).get("status") or "?"
    ok = fm["status_OK"] and fm["videoId"] and fm["mf_likeCount"]
    return r.status_code, len(body), ok, st, 1, fm


def integrity():
    """150 параллельных запросов одного видео: неизменные поля должны совпадать."""
    U.header("ЦЕЛОСТНОСТЬ (подделка / кэш)")
    V = "dQw4w9WgXcQ"

    def grab(_=None):
        try:
            h = U.S().get(f"https://m.youtube.com/watch?v={V}",
                          headers={"User-Agent": U.UA_MWEB},
                          impersonate=U.CFG["imp"], timeout=30).content
            b = U.blob(h)
            if not b: return None
            d = json.loads(b)
            vd = d.get("videoDetails", {})
            mf = d.get("microformat", {}).get("playerMicroformatRenderer", {})
            return (int(vd.get("viewCount",0)), int(mf.get("likeCount",0)),
                    vd.get("title"), int(vd.get("lengthSeconds",0)),
                    mf.get("publishDate"), vd.get("channelId"))
        except Exception:
            return None

    with concurrent.futures.ThreadPoolExecutor(24) as ex:
        vals = [v for v in ex.map(grab, range(150)) if v]
    if not vals:
        print("  нет данных"); return
    vw = [v[0] for v in vals]
    print(f"  n={len(vals)}  views spread={max(vw)-min(vw)} uniq={len(set(vw))}")
    bad = False
    for i, name in ((2,"title"),(3,"lengthSeconds"),(4,"publishDate"),(5,"channelId")):
        u = set(v[i] for v in vals)
        if len(u) > 1:
            bad = True
            print(f"  {name:14} uniq={len(u)}  <<< ИНВАРИАНТ НАРУШЕН  {list(u)[:3]}")
        else:
            print(f"  {name:14} ok")
    print("  ВЕРДИКТ:", "ПОДОЗРЕНИЕ НА ПОДДЕЛКУ" if bad else "данные консистентны")


if __name__ == "__main__":
    p = U.parser()
    p.add_argument("--integrity", action="store_true")
    a = p.parse_args()
    if a.integrity:
        if U.setup(a): integrity()
    else:
        U.run(a, "watch", do_watch, fields=True)
