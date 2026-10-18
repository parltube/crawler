"""
Проба комментариев: innertube /next за токеном секции комментариев,
затем /next с continuation за первой страницей. Один таргет = 2 запроса.

    python comments.py --label isp-tun --quick
"""
import utils as U


def do_comments(vid):
    ref = f"https://www.youtube.com/watch?v={vid}"
    r = U.post("next", {"context": U.ctx("WEB"), "videoId": vid}, "1", U.UA_WEB, ref)
    try:
        d = r.json()
    except Exception:
        return r.status_code, len(r.content), False, "bad_json", 0, {}
    tok = None
    for sec in U.find(d, "itemSectionRenderer"):
        if sec.get("sectionIdentifier") == "comment-item-section":
            cmd = U.find(sec, "continuationCommand")
            if cmd: tok = cmd[0].get("token")
            break
    if not tok:
        return r.status_code, len(r.content), False, "no_token", 0, {}

    r2 = U.post("next", {"context": U.ctx("WEB"), "continuation": tok}, "1", U.UA_WEB, ref)
    nb = len(r.content) + len(r2.content)
    try:
        d2 = r2.json()
    except Exception:
        return r2.status_code, nb, False, "bad_json2", 0, {}
    n = len(U.find(d2, "commentEntityPayload")) or len(U.find(d2, "commentRenderer"))
    if not n:
        return r2.status_code, nb, False, "no_comments", 0, {}
    return r2.status_code, nb, True, "OK", n, {}


if __name__ == "__main__":
    U.run(U.parser().parse_args(), "comments", do_comments, items="cmt")
