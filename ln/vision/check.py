"""Check a converted video against its source, frame by frame, and say where it goes wrong.

It works on pixels, so it can judge any converter: give it the footage and the drawing made from it.
For EVERY frame it measures

  flicker     how much the drawing changes beyond what the footage's own motion explains: the
              previous drawing frame is carried forward along the footage's optical flow and
              compared with this one (mean pixel difference, 0..255)
  fidelity    how close the drawing's broad colour and light are to the footage (PSNR of both
              blurred), so a frame that drifted off shows up
  structure   how much of the footage's strong edges (outlines, lane marks, limbs) the drawing
              still has a stroke or a colour change near: low means things went missing
  subjects    moving things are found from the flow; for each, the drawing's detail inside it
              is compared with the footage's. Low means that thing is under-drawn or has
              vanished, and a drop against its own neighbouring frames means it blinked out

and writes the numbers, the flagged frames and a picture of the worst ones.

    python ln/vision/check.py source.mp4 drawing.mp4 --report out/report.json --sheet out/worst.png
"""
import argparse
import json
import sys

import cv2
import numpy as np


def frames(path, width, fps):
    """Frames at exact times 0, 1/fps, 2/fps ... whatever the file's own frame rate is."""
    cap = cv2.VideoCapture(path)
    if not cap.isOpened():
        sys.exit("cannot open " + path)
    src_fps = cap.get(cv2.CAP_PROP_FPS) or fps
    i, want = 0, 0.0
    while True:
        ok, raw = cap.read()
        if not ok:
            return
        if i / src_fps >= want - 0.5 / src_fps:
            h = int(round(raw.shape[0] * width / raw.shape[1]))
            yield cv2.resize(raw, (width, h), interpolation=cv2.INTER_AREA)
            want += 1.0 / fps
        i += 1


def thumb(img):
    return cv2.resize(img, (32, 32), interpolation=cv2.INTER_AREA).astype(np.float32)


def is_cut(a, b):
    """A cut between two frames (their 32x32 thumbnails): the picture changed completely. A flash, a fade or a
    strobe changes brightness but not what is where, so the test compares the shapes (each frame's brightness
    pattern with its own average and spread removed), not the brightness itself."""
    ga, gb = a.mean(axis=2), b.mean(axis=2)
    sa, sb = float(ga.std()), float(gb.std())
    if sa < 3.0 or sb < 3.0:  # a blank or near-blank frame (a blackout) has no shapes to disagree with
        return False
    corr = float(((ga - ga.mean()) * (gb - gb.mean())).mean() / (sa * sb))
    ha = cv2.calcHist([a.astype(np.uint8)], [0, 1, 2], None, [6, 6, 6], [0, 256] * 3)
    hb = cv2.calcHist([b.astype(np.uint8)], [0, 1, 2], None, [6, 6, 6], [0, 256] * 3)
    cv2.normalize(ha, ha, 1, 0, cv2.NORM_L1)
    cv2.normalize(hb, hb, 1, 0, cv2.NORM_L1)
    diff = float(np.abs(a - b).mean())
    return corr < 0.2 and (diff > 18 or cv2.compareHist(ha, hb, cv2.HISTCMP_BHATTACHARYYA) > 0.5)


def warp_back(img, flow):
    """Pull `img` (the previous frame) onto the current frame along flow (current -> previous)."""
    h, w = flow.shape[:2]
    gx, gy = np.meshgrid(np.arange(w, dtype=np.float32), np.arange(h, dtype=np.float32))
    return cv2.remap(img, gx + flow[..., 0], gy + flow[..., 1], cv2.INTER_LINEAR, borderMode=cv2.BORDER_REPLICATE)


def edges_of(gray, strong):
    g = cv2.bilateralFilter(gray, 7, 40, 7)
    hi, _ = cv2.threshold(g, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)
    hi = max(30.0, hi * strong)
    return cv2.Canny(g, hi * 0.4, hi)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("source")
    ap.add_argument("drawing")
    ap.add_argument("--width", type=int, default=800)
    ap.add_argument("--fps", type=float, default=12)
    ap.add_argument("--report", default="out/report.json")
    ap.add_argument("--sheet", default=None, help="write a picture of the worst frames here")
    ap.add_argument("--worst", type=int, default=6)
    a = ap.parse_args()

    src_all = list(frames(a.source, a.width, a.fps))
    out_all = list(frames(a.drawing, a.width, a.fps))
    n = min(len(src_all), len(out_all))
    if abs(len(src_all) - len(out_all)) > 2:
        print(f"warning: {len(src_all)} source frames but {len(out_all)} drawing frames", file=sys.stderr)
    H, W = src_all[0].shape[:2]
    out_all = [cv2.resize(o, (W, H), interpolation=cv2.INTER_AREA) for o in out_all[:n]]
    src_all = src_all[:n]

    rows = []
    cut_at = [t > 0 and is_cut(thumb(src_all[t - 1]), thumb(src_all[t])) for t in range(n)]
    prev_gray = prev_out = None
    ratio_hist = {}  # subject track id -> list of (frame, relative detail)
    tracks = []  # (centroid, id) from the previous frame
    next_id = 0
    for t in range(n):
        src, out = src_all[t], out_all[t]
        gs = cv2.cvtColor(src, cv2.COLOR_BGR2GRAY)
        go = cv2.cvtColor(out, cv2.COLOR_BGR2GRAY)
        row = {"frame": t}
        # fidelity: broad colour and light
        bs, bo = cv2.GaussianBlur(src, (0, 0), 4), cv2.GaussianBlur(out, (0, 0), 4)
        mse = float(((bs.astype(np.float32) - bo.astype(np.float32)) ** 2).mean())
        row["fidelity_psnr"] = round(10 * np.log10(255 * 255 / max(mse, 1e-6)), 2)
        # structure: strong footage edges that still have something drawn near them
        es = edges_of(gs, 1.0)
        eo = edges_of(go, 0.6)
        dist = cv2.distanceTransform(255 - eo, cv2.DIST_L2, 3)
        n_es = int((es > 0).sum())
        row["structure_recall"] = round(float((dist[es > 0] <= 3).mean()) if n_es else 1.0, 3)
        subjects = []
        if cut_at[t]:  # a cut: nothing carries over, so there is nothing to compare with the last frame
            prev_gray = prev_out = None
            tracks = []
            row["cut"] = True
        if prev_gray is not None:
            small = (W // 2, H // 2)
            flow = cv2.calcOpticalFlowFarneback(cv2.resize(gs, small), cv2.resize(prev_gray, small), None, 0.5, 3, 15, 3, 5, 1.2, 0)
            flow = cv2.resize(flow, (W, H)) * 2.0
            warped = warp_back(prev_out, flow)
            diff = np.abs(out.astype(np.float32) - warped.astype(np.float32)).mean(axis=2)
            row["flicker"] = round(float(diff.mean()), 3)
            # subjects: things the flow says are moving on their own
            mag = np.hypot(flow[..., 0], flow[..., 1])
            med = float(np.median(mag))
            moving = ((mag - med) > 1.2).astype(np.uint8)
            moving = cv2.morphologyEx(moving, cv2.MORPH_OPEN, np.ones((5, 5), np.uint8))
            moving = cv2.dilate(moving, np.ones((9, 9), np.uint8))
            # where nothing moves the drawing should hold perfectly still: any change there is jitter
            still = cv2.erode(1 - cv2.dilate(moving, np.ones((15, 15), np.uint8)), np.ones((5, 5), np.uint8)).astype(bool)
            row["flicker_still"] = round(float(diff[still].mean()), 3) if still.sum() > 500 else None
            row["flicker_moving"] = round(float(diff[moving > 0].mean()), 3) if (moving > 0).sum() > 500 else None
            nc, lab, st, cen = cv2.connectedComponentsWithStats(moving, connectivity=8)
            glob_s, glob_o = max(float((es > 0).mean()), 1e-6), max(float((eo > 0).mean()), 1e-6)
            for c in range(1, nc):
                if st[c, cv2.CC_STAT_AREA] < 500:
                    continue
                m = lab == c
                ds, do = float((es[m] > 0).mean()), float((eo[m] > 0).mean())
                rel = (do / glob_o) / max(ds / glob_s, 1e-6)  # drawing's detail here vs its detail elsewhere, per footage detail
                # follow the subject from frame to frame by nearest centroid
                best, who = 1e9, None
                for (px, py), tid in tracks:
                    d = np.hypot(px - cen[c][0], py - cen[c][1])
                    if d < best and d < 90:
                        best, who = d, tid
                if who is None:
                    who, next_id = next_id, next_id + 1
                ratio_hist.setdefault(who, []).append((t, rel))
                subjects.append({"id": who, "x": int(st[c, 0]), "y": int(st[c, 1]), "w": int(st[c, 2]), "h": int(st[c, 3]), "area": int(st[c, 4]), "detail": round(rel, 2)})
            tracks = [((s["x"] + s["w"] / 2, s["y"] + s["h"] / 2), s["id"]) for s in subjects]
        else:
            row["flicker"] = 0.0
        row["subjects"] = subjects
        rows.append(row)
        prev_gray, prev_out = gs, out

    # subjects that dip against their own neighbours blinked out; subjects that are always low are under-drawn
    blinks, thin = [], []
    for tid, hist in ratio_hist.items():
        if len(hist) < 5:
            continue
        vals = np.array([r for _, r in hist])
        for k, (t, r) in enumerate(hist):
            around = np.concatenate([vals[max(0, k - 3):k], vals[k + 1:k + 4]])
            if len(around) >= 3 and r < 0.5 * np.median(around) and np.median(around) > 0.4:
                blinks.append({"frame": t, "subject": tid, "detail": round(float(r), 2), "neighbours": round(float(np.median(around)), 2)})
        if float(np.median(vals)) < 0.45:
            thin.append({"subject": tid, "frames": [hist[0][0], hist[-1][0]], "median_detail": round(float(np.median(vals)), 2)})

    smooth_rows = [r for r in rows if r["frame"] > 0 and not r.get("cut")]  # flicker only means something between frames of one shot
    fl_s = np.array([r["flicker"] for r in smooth_rows]) if smooth_rows else np.zeros(1)
    fl = np.array([r["flicker"] for r in rows])
    fid = np.array([r["fidelity_psnr"] for r in rows])
    rec = np.array([r["structure_recall"] for r in rows])
    flick_cut = float(np.median(fl_s) * 1.6)
    flagged = [r["frame"] for r in rows if r["frame"] and not r.get("cut") and (r["flicker"] > flick_cut or r["structure_recall"] < 0.55)]
    still_v = [r["flicker_still"] for r in rows[1:] if r.get("flicker_still") is not None]
    move_v = [r["flicker_moving"] for r in rows[1:] if r.get("flicker_moving") is not None]
    summary = {
        "frames": n,
        "cuts": int(sum(cut_at)),
        "flicker_still_mean": round(float(np.mean(still_v)), 3) if still_v else None,
        "flicker_moving_mean": round(float(np.mean(move_v)), 3) if move_v else None,
        "flicker_mean": round(float(fl_s.mean()), 3),
        "flicker_p95": round(float(np.percentile(fl_s, 95)), 3),
        "fidelity_psnr_mean": round(float(fid.mean()), 2),
        "structure_recall_mean": round(float(rec.mean()), 3),
        "structure_recall_min": round(float(rec.min()), 3),
        "blinks": len(blinks),
        "under_drawn_subjects": len(thin),
        "flagged_frames": len(flagged),
    }
    json.dump({"summary": summary, "frames": rows, "blinks": blinks, "under_drawn": thin, "flagged": flagged}, open(a.report, "w"), indent=1)
    print(json.dumps(summary, indent=1))
    if blinks:
        print("blinks (a subject's detail dips against its neighbouring frames):", blinks[:8], file=sys.stderr)
    if thin:
        print("under-drawn subjects:", thin[:8], file=sys.stderr)

    if a.sheet:
        score = [(r["flicker"] / max(np.median(fl[1:]), 1e-6)) + (1 - r["structure_recall"]) * 4 for r in rows]
        worst = sorted(np.argsort(score)[::-1][: a.worst])
        tiles = []
        for t in worst:
            both = np.hstack([cv2.resize(src_all[t], (W // 2, H // 2)), cv2.resize(out_all[t], (W // 2, H // 2))])
            cv2.putText(both, f"frame {t} flicker {rows[t]['flicker']:.1f} recall {rows[t]['structure_recall']:.2f}", (6, 16), cv2.FONT_HERSHEY_SIMPLEX, 0.45, (0, 0, 255), 1)
            for s in rows[t]["subjects"]:
                cv2.rectangle(both, (W // 2 + s["x"] // 2, s["y"] // 2), (W // 2 + (s["x"] + s["w"]) // 2, (s["y"] + s["h"]) // 2), (0, 160, 255) if s["detail"] > 0.45 else (0, 0, 255), 1)
            tiles.append(both)
        cols = 2
        while len(tiles) % cols:
            tiles.append(np.zeros_like(tiles[0]))
        grid = np.vstack([np.hstack(tiles[i:i + cols]) for i in range(0, len(tiles), cols)])
        cv2.imwrite(a.sheet, grid)


if __name__ == "__main__":
    main()
