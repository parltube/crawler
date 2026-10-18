"""
Проба innertube /player (клиент MWEB).

    python player.py --label isp-tun --quick
"""
import utils as U


def do_player(vid):
    r = U.post("player", {"context": U.ctx("MWEB"), "videoId": vid}, "2", U.UA_MWEB,
               f"https://www.youtube.com/watch?v={vid}")
    try:
        d = r.json()
    except Exception:
        return r.status_code, len(r.content), False, "bad_json", 0, U.EMPTY
    ps = d.get("playabilityStatus") or {}
    st = ps.get("status") or "?"
    if st != "OK":
        rs = (ps.get("reason") or "")[:24].replace(" ", "_")
        return r.status_code, len(r.content), False, f"{st}:{rs}", 0, U.EMPTY
    fm = U.fmap(d)
    return r.status_code, len(r.content), fm["videoId"], "OK", 1, fm


if __name__ == "__main__":
    U.run(U.parser().parse_args(), "player", do_player, fields=True)
