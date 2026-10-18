"""
Проба innertube /browse (вкладка Videos канала).

    python browse.py --label isp-tun --quick
"""
import utils as U

CHANS = ["UCX6OQ3DkcsbYNE6H8uQQuVA","UCuAXFkgsw1L7xaCfnd5JJOw",
         "UC-9-kyTW8ZkZNDHQJ6FgpwQ","UCq-Fj5jknLsUf-MWSy4_brA",
         "UCbCmjCuTUZos6Inko4u57UQ","UCBJycsmduvYEL83R_U4JriQ"]


def do_browse(ch):
    r = U.post("browse", {"context": U.ctx("WEB"), "browseId": ch,
                          "params": "EgZ2aWRlb3PyBgQKAjoA"}, "1", U.UA_WEB,
               f"https://www.youtube.com/channel/{ch}/videos")
    try:
        d = r.json()
    except Exception:
        return r.status_code, len(r.content), False, "bad_json", 0, {}
    if "captchaHandler" in r.text:
        return r.status_code, len(r.content), False, "captcha", 0, {}
    lus = U.find(d, "lockupViewModel")
    if not lus:
        note = "alert" if '"alerts"' in r.text else "no_videos"
        return r.status_code, len(r.content), False, note, 0, {}
    return r.status_code, len(r.content), True, "OK", len(lus), {}


if __name__ == "__main__":
    U.run(U.parser().parse_args(), "browse", do_browse, targets=CHANS, items="vid")
