"""Turn any video into flat-colour patches and ink lines that stay put from frame to frame.

Plain classical OpenCV, NumPy and scikit-image, CPU only, no trained models, nothing specific to any video.

Every frame goes through the same steps, and everything that changes over time is carried forward
along the footage's optical flow, so the drawing moves with the picture instead of boiling:

  1. look      mean-shift flattening; CLAHE lifts detail out of shadows
  2. palette   one k-means palette for the whole clip (colourfulness weighted up), plus a skin class
  3. label     every pixel takes its palette colour; last frame's labels, moved along the flow, vote too
  4. merge     neighbouring patches whose shared border has no real edge are one surface and join
  5. absorb    small patches fold into the neighbour around them (never dropped, so no holes);
               what counts as small is lower where things are moving (the subject keeps its detail)
  6. track     each patch inherits the identity of the patch it was last frame
  7. lines     drawn like an artist: each border once (silhouette), inner edges nothing covers (form),
               thin shading where the light really changes; busy texture left as flat tone; anything
               that stands out or moves is always outlined

One JSON object per line (one per frame) on stdout, or into a file:

  {"i": 0, "w": 640, "h": 360,
   "regions": [{"c": [r, g, b], "area": 123, "x": contrast, "id": 7, "loops": [[[x, y], ...], ...]}, ...],
   "lines":   [{"k": 0, "l": 0.4, "w": 2.1, "p": [[x, y], ...]}, ...]}

Coordinates are pixels, y down. Colours are 0..255 RGB. Regions are largest first. "x" is how different
the colours around a patch are (low: a soft edge). In lines, k is the pen: 0 silhouette (the border
between two patches, once), 1 form (an inner edge nothing covers), 2 shade (a thin brightness contour,
l is the brightness level, 0..1); w is the line's width in pixels.
"""
import argparse
import json
import sys

import cv2
import numpy as np
from skimage import measure

CHROMA = 1.9  # colourfulness counts this much more than brightness when picking and matching colours


# ---------------------------------------------------------------- reading


def read_frames(path, width, fps):
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


# ---------------------------------------------------------------- looking


CLAHE = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(6, 6))


def lift_shadows(gray):
    """Half the picture's own brightness, half locally equalised: detail inside shadows separates."""
    return cv2.addWeighted(gray, 0.5, CLAHE.apply(gray), 0.5, 0)


def to_lab(frame):
    lab = cv2.cvtColor(frame, cv2.COLOR_BGR2LAB).astype(np.float32)
    lab[..., 1:] = 128 + (lab[..., 1:] - 128) * CHROMA
    lab[..., 0] = 0.5 * lab[..., 0] + 0.5 * CLAHE.apply(np.clip(lab[..., 0], 0, 255).astype(np.uint8)).astype(np.float32)
    return lab


def flatten(frame, spatial, colour):
    return cv2.pyrMeanShiftFiltering(frame, spatial, colour, maxLevel=1)


def skin_mask(flat):
    """Skin-coloured pixels, by a colour rule that holds for light and dark skin alike."""
    ycc = cv2.cvtColor(flat, cv2.COLOR_BGR2YCrCb)
    hsv = cv2.cvtColor(flat, cv2.COLOR_BGR2HSV)
    cr, cb = ycc[..., 1], ycc[..., 2]
    m = ((cr > 140) & (cr < 175) & (cb > 82) & (cb < 122) & (hsv[..., 1] > 55) & (hsv[..., 1] < 190) & (hsv[..., 2] > 45)).astype(np.uint8)
    m = cv2.morphologyEx(m, cv2.MORPH_OPEN, np.ones((3, 3), np.uint8))
    m = cv2.morphologyEx(m, cv2.MORPH_CLOSE, cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (7, 7)))
    # faces and hands are small: a skin-coloured patch bigger than this is clothing, wood or floor
    n, lab, st, _ = cv2.connectedComponentsWithStats(m, connectivity=8)
    big = m.shape[0] * m.shape[1] * 0.025
    for c in range(1, n):
        if st[c, cv2.CC_STAT_AREA] > big or st[c, cv2.CC_STAT_AREA] < 40:
            m[lab == c] = 0
    return m


# ---------------------------------------------------------------- palette


def merge_close(centres, limit):
    """Fold together palette colours closer than `limit`, so one face is not five skin tones."""
    cs = [c.astype(np.float64) for c in centres]
    while len(cs) > 2:
        best, pair = limit, None
        for i in range(len(cs)):
            for j in range(i + 1, len(cs)):
                d = float(np.linalg.norm(cs[i] - cs[j]))
                if d < best:
                    best, pair = d, (i, j)
        if pair is None:
            break
        i, j = pair
        cs[i] = (cs[i] + cs[j]) / 2
        del cs[j]
    return np.array(cs, np.float32)


def lift(frame, target=72.0):
    """Brighten a dark picture (never darkens) so shapes in shadow still separate into patches: first stretch
    its own range over the whole scale, if it only uses a small part of it, then a gamma toward `target`."""
    hi = float(np.percentile(frame.max(axis=2), 99.5))  # the brightest channel, so pure red is not "dim"
    if hi < 110:  # a dim picture: scale every channel by the same amount, so colours stay as they were
        frame = np.clip(frame.astype(np.float32) * (180.0 / max(hi, 12.0)), 0, 255).astype(np.uint8)
    m = float(cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY).mean())
    if m >= target:
        return frame
    g = max(0.6, np.log(target / 255.0) / np.log(max(m, 4.0) / 255.0))
    table = (255.0 * (np.arange(256) / 255.0) ** g).astype(np.uint8)
    return cv2.LUT(frame, table)


def tiny(frame):
    return cv2.resize(frame, (32, 32), interpolation=cv2.INTER_AREA).astype(np.float32)


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


def palette_from(frames, colours, merge):
    """k-means over the pixels of a few frames: the palette of one shot."""
    pix, skin = [], []
    for frame in frames:
        frame = lift(frame)
        flat = flatten(frame, 8, 20)
        lab = to_lab(flat).reshape(-1, 3)
        pix.append(lab[:: max(1, len(lab) // 5000)])
        s = skin_mask(flat).reshape(-1) > 0
        if s.any():
            skin.append(lab[s][:: max(1, int(s.sum()) // 1500)])
    data = np.vstack(pix).astype(np.float32)
    k = max(2, min(colours, len(data) // 400))
    crit = (cv2.TERM_CRITERIA_EPS + cv2.TERM_CRITERIA_MAX_ITER, 30, 0.5)
    _, _, centres = cv2.kmeans(data, k, None, crit, 3, cv2.KMEANS_PP_CENTERS)
    centres = merge_close(centres, merge)
    skin_lab = None
    if skin and sum(len(x) for x in skin) > 600:
        skin_lab = np.vstack(skin).mean(axis=0).astype(np.float32)
    return centres, skin_lab


def make_palette(path, width, fps, colours, merge, every=12):
    """k-means over pixels from frames spread through the video (one in every `every`)."""
    pix, skin, n, cuts, last = [], [], 0, 0, None
    for i, frame in enumerate(read_frames(path, width, fps)):
        n = i + 1
        small = tiny(frame)
        if last is not None and is_cut(last, small):
            cuts += 1
        last = small
        if i % every:
            continue
        frame = lift(frame)
        flat = flatten(frame, 8, 20)
        lab = to_lab(flat).reshape(-1, 3)
        pix.append(lab[:: max(1, len(lab) // 6000)])
        s = skin_mask(flat).reshape(-1) > 0
        if s.any():
            skin.append(lab[s][:: max(1, int(s.sum()) // 2000)])
    if not pix:
        sys.exit("no frames in " + path)
    data = np.vstack(pix).astype(np.float32)
    crit = (cv2.TERM_CRITERIA_EPS + cv2.TERM_CRITERIA_MAX_ITER, 30, 0.5)
    _, _, centres = cv2.kmeans(data, colours, None, crit, 4, cv2.KMEANS_PP_CENTERS)
    centres = merge_close(centres, merge)
    skin_lab = None
    if skin and sum(len(x) for x in skin) > 800:  # enough skin to be worth its own colour
        skin_lab = np.vstack(skin).mean(axis=0).astype(np.float32)
    return centres, skin_lab, n, cuts


def to_rgb(lab_centre):
    c = np.array(lab_centre, np.float32)
    c[1:] = 128 + (c[1:] - 128) / CHROMA
    px = np.uint8([[np.clip(c, 0, 255)]])
    b, g, r = cv2.cvtColor(px, cv2.COLOR_LAB2BGR)[0, 0]
    return [int(r), int(g), int(b)]


def label_frame(lab, centres):
    d = ((lab[:, :, None, :] - centres[None, None, :, :]) ** 2).sum(axis=3)
    return d.argmin(axis=2)


# ---------------------------------------------------------------- moving things along the flow


def flow_between(gray, prev_gray):
    """Where each pixel of this frame was in the last one (x, y), at full size."""
    h, w = gray.shape
    small = (w // 2, h // 2)
    f = cv2.calcOpticalFlowFarneback(cv2.resize(gray, small), cv2.resize(prev_gray, small), None, 0.5, 3, 15, 3, 5, 1.2, 0)
    return cv2.resize(f, (w, h)) * 2.0


def steady(flow):
    """Flow with the noise taken out: smoothed, and zero where it is too small to be real motion.
    Noise in flat areas would otherwise shift the carried-forward picture a little every frame."""
    f = cv2.GaussianBlur(flow, (0, 0), 6)
    f[np.hypot(f[..., 0], f[..., 1]) < 0.4] = 0
    return f


def denoise(frame, prev_clean, flow, strength=0.8, trust=22.0):
    """Average each pixel with the same spot in the cleaned last frame, moved along the flow, wherever
    the two agree. Camera and compression noise differ from frame to frame; real things do not."""
    f = frame.astype(np.float32)
    if prev_clean is None or flow is None:
        return f
    before = warp(prev_clean, flow)
    d = np.abs(f - before).max(axis=2)
    w = (np.clip(1.0 - d / trust, 0.0, 1.0) * strength)[..., None]
    return w * before + (1.0 - w) * f


def hold(prob, prev_final, flow, bonus, imp):
    """The label of each pixel: the likeliest, but a pixel keeps the patch it was in last frame
    (moved along the flow) unless another wins by a clear margin: a patch whose footage colour does
    not change never changes colour, however the camera noise wobbles. Where things are moving the
    margin is smaller, so a real change (a person stepping in, the light turning) still gets through."""
    logits = prob.copy()
    if prev_final is not None and flow is not None:
        before = warp(prev_final.astype(np.float32), flow, nearest=True).astype(np.int64)[..., None]
        margin = (bonus * (1.0 - 0.7 * imp))[..., None]
        np.put_along_axis(logits, before, np.take_along_axis(logits, before, 2) + margin, 2)
    return logits.argmax(axis=2).astype(np.uint8)


def memory(calm, imp):
    """How much of the last frame to keep at each pixel: a lot where nothing moves, less on moving things."""
    return (calm * (1.0 - 0.55 * imp)).astype(np.float32)


def warp(img, flow, nearest=False):
    h, w = img.shape[:2]
    gx, gy = np.meshgrid(np.arange(w, dtype=np.float32), np.arange(h, dtype=np.float32))
    mx, my = gx + flow[..., 0], gy + flow[..., 1]
    interp = cv2.INTER_NEAREST if nearest else cv2.INTER_LINEAR
    if img.ndim == 2 or img.shape[2] <= 4:
        return cv2.remap(img, mx, my, interp, borderMode=cv2.BORDER_REPLICATE)
    out = np.empty_like(img)
    for k in range(img.shape[2]):
        out[..., k] = cv2.remap(img[..., k], mx, my, interp, borderMode=cv2.BORDER_REPLICATE)
    return out


def importance(flow, prev):
    """0..1: how much the footage is moving on its own here (camera motion is the median)."""
    mag = np.hypot(flow[..., 0], flow[..., 1])
    m = np.clip((mag - float(np.median(mag)) - 0.8) / 2.5, 0, 1)
    m = cv2.GaussianBlur(m.astype(np.float32), (0, 0), 10)
    return m if prev is None else np.maximum(m, 0.6 * prev)


def texture(gray):
    """0..1: how busy the picture is here (leaves, gravel): local spread of brightness."""
    g = gray.astype(np.float32) / 255.0
    mean = cv2.GaussianBlur(g, (0, 0), 4)
    var = np.maximum(cv2.GaussianBlur(g * g, (0, 0), 4) - mean * mean, 0)
    return np.clip(np.sqrt(var) / 0.12, 0, 1)


# ---------------------------------------------------------------- patches


def components(label, k_count):
    """Connected patches of equal label, numbered across all labels. Returns (ids, label of each, area of each)."""
    ids = np.zeros(label.shape, np.int32)
    lab_of, area_of, base = [0], [0], 1
    for k in range(k_count):
        n, cm, st, _ = cv2.connectedComponentsWithStats((label == k).astype(np.uint8), connectivity=4)
        if n <= 1:
            continue
        sel = cm > 0
        ids[sel] = cm[sel] + base - 1
        lab_of.extend([k] * (n - 1))
        area_of.extend(st[1:, cv2.CC_STAT_AREA].tolist())
        base += n - 1
    return ids, np.array(lab_of), np.array(area_of, np.float64)


def merge_weak(label, centres, gray, k_count, border, limit, lab_img):
    """Join neighbouring patches whose shared border has no real edge on it: one surface, cut in two
    by a palette step across a smooth gradient. Colours must also be close, so a chain of merges
    cannot slide from one thing into another: a group keeps the range of colours it holds, and no channel of
    that range may grow wider than `limit`, so a chain of small steps cannot carry one colour all the way to
    another."""
    ids, lab_of, area = components(label, k_count)
    g = cv2.GaussianBlur(gray.astype(np.float32), (0, 0), 1.2)
    gx, gy = cv2.Sobel(g, cv2.CV_32F, 1, 0), cv2.Sobel(g, cv2.CV_32F, 0, 1)
    mag = np.hypot(gx, gy) / 8.0  # roughly: brightness steps across one pixel
    n = len(lab_of)
    grp_mean = np.stack([np.bincount(ids.ravel(), weights=lab_img[..., c].ravel(), minlength=n) for c in range(3)], axis=1) / np.maximum(area, 1)[:, None]
    grp_lo, grp_hi = grp_mean.copy(), grp_mean.copy()  # each group's colour range, so it can never drift
    pairs = {}
    for a, b, m in ((ids[:, :-1], ids[:, 1:], np.maximum(mag[:, :-1], mag[:, 1:])), (ids[:-1, :], ids[1:, :], np.maximum(mag[:-1, :], mag[1:, :]))):
        diff = a != b
        if not diff.any():
            continue
        lo, hi = np.minimum(a[diff], b[diff]).astype(np.int64), np.maximum(a[diff], b[diff]).astype(np.int64)
        key = lo * n + hi
        uk, inv = np.unique(key, return_inverse=True)
        sums = np.bincount(inv, weights=m[diff])
        cnt = np.bincount(inv)
        for u, s, c in zip(uk, sums, cnt):
            old = pairs.get(int(u), (0.0, 0))
            pairs[int(u)] = (old[0] + float(s), old[1] + int(c))
    parent = list(range(n))

    def find(x):
        while parent[x] != x:
            parent[x] = parent[parent[x]]
            x = parent[x]
        return x

    order = sorted(((s / c, int(k // n), int(k % n)) for k, (s, c) in pairs.items() if c >= 3), key=lambda t: t[0])
    for strength, a, b in order:
        if strength > border:
            break
        if a == 0 or b == 0:
            continue
        ra, rb = find(a), find(b)
        if ra == rb:
            continue
        new_lo, new_hi = np.minimum(grp_lo[ra], grp_lo[rb]), np.maximum(grp_hi[ra], grp_hi[rb])
        if float((new_hi - new_lo).max()) > limit:
            continue  # joining would put clearly different colours in one patch
        if area[ra] < area[rb]:
            ra, rb = rb, ra
        parent[rb] = ra
        area[ra] += area[rb]
        grp_lo[ra], grp_hi[ra] = new_lo, new_hi
    root = np.array([find(i) for i in range(n)])
    return lab_of[root][ids].astype(label.dtype)


def absorb_small(label, k_count, min_area, imp, skin_k=-1, centres=None, stand_out=45.0):
    """Give every patch smaller than its threshold to the label around it most, so nothing is dropped
    and no holes show paper. The threshold falls where things are moving: the subject keeps detail."""
    for _ in range(2):
        for k in range(k_count):
            n, lab, st, cen = cv2.connectedComponentsWithStats((label == k).astype(np.uint8), connectivity=8)
            for c in range(1, n):
                area = st[c, cv2.CC_STAT_AREA]
                if k == skin_k:
                    continue
                cx, cy = int(cen[c][0]), int(cen[c][1])
                limit = min_area * (1.0 - 0.8 * float(imp[min(cy, imp.shape[0] - 1), min(cx, imp.shape[1] - 1)]))
                if area >= limit:
                    continue
                x, y, w, h = st[c, :4]
                y0, y1, x0, x1 = max(0, y - 2), min(label.shape[0], y + h + 2), max(0, x - 2), min(label.shape[1], x + w + 2)
                comp = lab[y0:y1, x0:x1] == c
                ring = cv2.dilate(comp.astype(np.uint8), np.ones((3, 3), np.uint8)).astype(bool) & ~comp
                near = label[y0:y1, x0:x1][ring]
                if len(near) == 0:
                    continue
                count = np.bincount(near, minlength=k_count)
                count[k] = 0
                into = int(count.argmax())
                if into == skin_k and area >= limit * 0.12:
                    continue  # a feature of the face or hand (an eye, a brow): keep it
                if centres is not None and area >= 24 and float(np.linalg.norm(centres[k] - centres[into])) >= stand_out:
                    continue  # small but unlike what is around it (a cone on a road, a lamp): an object, keep it
                label[y0:y1, x0:x1][comp] = into
    return label


def contrast_of(label, mask_crop, box, centres, k):
    """How different the colours around a patch are from the patch: low means a soft edge."""
    y0, y1, x0, x1 = box
    ring = cv2.dilate(mask_crop.astype(np.uint8), np.ones((5, 5), np.uint8)).astype(bool) & ~mask_crop
    near = label[y0:y1, x0:x1][ring]
    if len(near) == 0:
        return 0.0
    count = np.bincount(near, minlength=len(centres)).astype(np.float64)
    count[k] = 0
    if count.sum() == 0:
        return 0.0
    dist = np.linalg.norm(centres - centres[k], axis=1)
    return float((count * dist).sum() / count.sum())


def shapes(label, k, centres, min_area, eps, frame, pids):
    """Connected patches of one label as outlines with their holes, each with its contrast, the footage's own
    mean colour over its whole area (holes left out), and the identity most of it already had."""
    mask = (label == k).astype(np.uint8)
    cnts, hier = cv2.findContours(mask, cv2.RETR_CCOMP, cv2.CHAIN_APPROX_NONE)
    if hier is None:
        return []
    hier = hier[0]
    out = []
    for i, c in enumerate(cnts):
        if hier[i][3] != -1:  # a hole: handled with its parent
            continue
        area = cv2.contourArea(c)
        if area < min_area * 0.2:
            continue
        loops = [cv2.approxPolyDP(c, eps, True)[:, 0, :]]
        holes = []
        child = hier[i][2]
        while child != -1:
            if cv2.contourArea(cnts[child]) >= min_area * 0.1:
                loops.append(cv2.approxPolyDP(cnts[child], eps, True)[:, 0, :])
            holes.append(cnts[child])
            child = hier[child][0]
        loops = [l.tolist() for l in loops if len(l) >= 3]
        if not loops:
            continue
        x, y, w, h = cv2.boundingRect(c)
        box = (max(0, y - 3), min(label.shape[0], y + h + 3), max(0, x - 3), min(label.shape[1], x + w + 3))
        off = -np.array([box[2], box[0]])
        m = np.zeros((box[1] - box[0], box[3] - box[2]), np.uint8)
        cv2.drawContours(m, [c + off], -1, 1, -1)
        contrast = contrast_of(label, m.astype(bool), box, centres, k)
        for hc in holes:
            cv2.drawContours(m, [hc + off], -1, 0, -1)
        b, g, r, _ = cv2.mean(frame[box[0]:box[1], box[2]:box[3]], mask=m)
        votes = pids[box[0]:box[1], box[2]:box[3]][m > 0]
        votes = votes[votes > 0]
        pid = int(np.bincount(votes).argmax()) if len(votes) else 0
        out.append((area, loops, contrast, (r, g, b), pid))
    return out


def give_ids(label, flow, prev_ids, next_id):
    """Each patch takes the identity of the patch most of it was last frame (following the flow);
    a patch with no ancestor, or whose ancestor went to a bigger patch, is new."""
    ids, _, area = components(label, int(label.max()) + 1)
    n = len(area)
    out = np.zeros(n, np.int64)
    if prev_ids is not None and flow is not None:
        before = warp(prev_ids.astype(np.float32), flow, nearest=True).astype(np.int64)
        claims = {}
        for c in range(1, n):
            votes = before[ids == c]
            votes = votes[votes > 0]
            if len(votes) == 0:
                continue
            vals, counts = np.unique(votes, return_counts=True)
            j = counts.argmax()
            if counts[j] >= 0.35 * area[c]:
                claims.setdefault(int(vals[j]), []).append(c)
        for pid, cs in claims.items():
            out[max(cs, key=lambda c: area[c])] = pid  # the biggest claimant keeps the identity
    for c in range(1, n):
        if out[c] == 0:
            next_id += 1
            out[c] = next_id
    return ids, out, next_id


# ---------------------------------------------------------------- lines


def contour_lines(field, level, imp, tex, a, kind, w, h, grad=None):
    """Lines where `field` crosses `level`, traced to a fraction of a pixel (marching squares), so a line
    glides with the field instead of stepping a pixel at a time. Scraps that are noise are dropped, and
    so are brightness contours that wander through flat areas: with `grad` (how fast the brightness
    changes at each pixel), a line is kept only if the light really changes along it."""
    lines = []
    for c in measure.find_contours(field, level):  # (row, col); lines that leave the frame simply end there
        if len(c) < 6:
            continue
        q = np.stack([c[:, 1] + 0.5, c[:, 0] + 0.5], axis=1).astype(np.float32)  # pixel centres
        q = cv2.approxPolyDP(q.reshape(-1, 1, 2), a.line_smooth, False)[:, 0, :]
        if len(q) < 2:
            continue
        length = float(np.hypot(*np.diff(q, axis=0).T).sum())
        size = float(max(np.ptp(q[:, 0]), np.ptp(q[:, 1])))
        cx, cy = int(np.clip(q[:, 0].mean(), 0, w - 1)), int(np.clip(q[:, 1].mean(), 0, h - 1))
        scale = (1.0 - 0.65 * float(imp[cy, cx])) * (1.0 + 1.4 * float(tex[cy, cx]) * (1.0 - float(imp[cy, cx])))
        if length < a.min_len * scale or size < a.min_size * scale:
            continue
        if tex[cy, cx] > a.busy and imp[cy, cx] < 0.3 and length < a.min_len * 4:
            continue  # a squiggle in leaves or gravel: the texture stays flat, only long lines survive
        if grad is not None:
            gx = np.clip(q[:, 0].astype(int), 0, w - 1)
            gy = np.clip(q[:, 1].astype(int), 0, h - 1)
            if float(grad[gy, gx].mean()) < a.min_grad * (1.0 - 0.6 * float(imp[cy, cx])):
                continue  # a contour through a flat area: it traces noise, not anything in the picture
        lines.append({"k": kind, "l": round(level, 3), "p": np.round(q, 2).tolist()})
    return lines


class Pencil:
    """Draws the lines of one frame the way an artist would: each real boundary once, in the right pen.

      pen 0  silhouette  the border between two patches, once (never twice, though both patches own it),
                         heavier the more the two differ; always drawn for things that stand out
      pen 1  form        a strong edge inside a patch that no silhouette already covers
      pen 2  shade       a thin line of the brightness inside a shaded form, only where the light really
                         changes and nothing is drawn already

    Busy texture (grass, gravel, leaves) gets no line per blade: it stays a flat tone. A line is kept only
    where nothing is drawn within a couple of pixels, so edges are never doubled."""

    def __init__(self, h, w, imp, tex, a):
        self.h, self.w, self.imp, self.tex, self.a = h, w, imp, tex, a
        self.drawn = np.zeros((h, w), np.uint8)
        self.dt = np.full((h, w), 99.0, np.float32)
        self.lines = []

    def _at(self, arr, pts):
        xs = np.clip(np.round(pts[:, 0]).astype(int), 0, self.w - 1)
        ys = np.clip(np.round(pts[:, 1]).astype(int), 0, self.h - 1)
        return arr[ys, xs]

    def refresh(self):
        self.dt = cv2.distanceTransform(255 - self.drawn, cv2.DIST_L2, 3)

    def runs(self, pts, gap=1.8):
        """The stretches of a line with nothing drawn beside them yet."""
        fresh = self._at(self.dt, pts) > gap
        out, cur = [], []
        for p, f in zip(pts, fresh):
            if f:
                cur.append(p)
            elif cur:
                out.append(np.array(cur))
                cur = []
        if cur:
            out.append(np.array(cur))
        return [r for r in out if len(r) >= 2]

    def add(self, pts, pen, width, level=0.0):
        q = pts.astype(np.float32)
        cv2.polylines(self.drawn, [np.round(q).astype(np.int32)], False, 255, 1)
        self.lines.append({"k": pen, "l": round(level, 3), "w": round(float(width), 2), "p": np.round(q, 2).tolist()})

    def silhouettes(self, regions):
        a = self.a
        for r in regions:  # largest first, so a big patch's border is drawn once before its small neighbours
            contrast = r["x"]
            cx, cy = r["_c"]
            im, tx = float(self.imp[cy, cx]), float(self.tex[cy, cx])
            standout = contrast >= a.stand_out or im > 0.25 or r["_sat"] > 70
            if contrast < a.sil_contrast and not standout:
                continue
            for loop in r["loops"]:
                pts = np.array(loop + [loop[0]], np.float32) + 0.5
                for run in self.runs(pts):
                    mid = run[len(run) // 2]
                    mx, my = int(np.clip(mid[0], 0, self.w - 1)), int(np.clip(mid[1], 0, self.h - 1))
                    busy = self.tex[my, mx] > 0.55 and self.imp[my, mx] < 0.3
                    if busy and not standout:
                        continue  # a blade of grass is not a line
                    length = float(np.hypot(*np.diff(run, axis=0).T).sum())
                    if length < a.min_len * (0.5 if standout else 1.0) * (1.0 - 0.5 * float(self.imp[my, mx])):
                        continue
                    self.add(cv2.approxPolyDP(run.reshape(-1, 1, 2), a.line_smooth, False)[:, 0, :], 0, 1.3 + 2.1 * float(np.clip(contrast / 60.0, 0, 1)))
            self.refresh()

    def layer(self, found, pen, width_of):
        """Add lines that are not already drawn; `found` come from contour_lines."""
        a = self.a
        for l in found:
            pts = np.array(l["p"], np.float32)
            for run in self.runs(pts, 2.2):
                length = float(np.hypot(*np.diff(run, axis=0).T).sum())
                mid = run[len(run) // 2]
                mx, my = int(np.clip(mid[0], 0, self.w - 1)), int(np.clip(mid[1], 0, self.h - 1))
                if length < a.min_len * (1.0 - 0.5 * float(self.imp[my, mx])):
                    continue
                self.add(run, pen, width_of(l), l["l"])
        self.refresh()


# ---------------------------------------------------------------- main


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("video")
    ap.add_argument("--out", default="-", help="file to write (default stdout)")
    ap.add_argument("--width", type=int, default=640, help="working width in pixels")
    ap.add_argument("--fps", type=float, default=12)
    ap.add_argument("--colours", type=int, default=14, help="size of the palette")
    ap.add_argument("--merge", type=float, default=26, help="fold palette colours closer than this together")
    ap.add_argument("--flat", type=float, default=18, help="colour radius of the flattening: higher gives flatter patches")
    ap.add_argument("--calm", type=float, default=0.85, help="0..1 how much of the last frame (moved along the flow) to keep where nothing moves; less on moving things")
    ap.add_argument("--min-area", type=float, default=220, help="patches smaller than this many pixels fold into their neighbour")
    ap.add_argument("--hold", type=float, default=1.0, help="how clearly a new patch must win before a pixel leaves the one it was in (stops borders flickering)")
    ap.add_argument("--border", type=float, default=5.0, help="patches meeting along an edge weaker than this join")
    ap.add_argument("--smooth", type=float, default=1.0, help="outline simplification of colour patches in pixels")
    ap.add_argument("--line-smooth", type=float, default=0.6, help="simplification of lines in pixels")
    ap.add_argument("--style", default="contour", choices=["contour", "pencil"], help="contour: thin lines from the footage's light and edges (default); pencil: one line per border between colour patches")
    ap.add_argument("--clutter", type=float, default=45, help="percentile of edge density above which a place counts as busy and is blurred much harder (lower flattens more of the picture)")
    ap.add_argument("--dense", type=float, default=100, help="percentile of line density above which short lines are dropped as clutter (lower drops more)")
    ap.add_argument("--busy", type=float, default=0.5, help="texture (0..1) above which short squiggles are dropped: leaves and gravel stay flat")
    ap.add_argument("--sil-contrast", type=float, default=24, help="two patches must differ this much (0..~100) to get a silhouette line between them")
    ap.add_argument("--stand-out", type=float, default=45, help="a patch that differs this much from its surroundings is always outlined, even in busy texture")
    ap.add_argument("--tones", type=int, default=4, help="brightness contours to draw")
    ap.add_argument("--blur", type=float, default=2.0, help="blur radius before tracing brightness (more: bigger simpler shapes)")
    ap.add_argument("--edge-cut", type=float, default=0.65, help="how strong an edge must be to be drawn, 0..1 (lower draws more)")
    ap.add_argument("--merge-colour", type=float, default=34, help="a merged patch may hold colours no further apart than this in any channel (higher: bigger flatter patches)")
    ap.add_argument("--detail-area", type=float, default=70, help="smallest detail patch in pixels")
    ap.add_argument("--detail-min", type=float, default=30, help="a detail patch must differ this much (RGB distance) from the patch under it")
    ap.add_argument("--detail-max", type=int, default=160, help="most detail patches per frame (0 = none)")
    ap.add_argument("--min-grad", type=float, default=0.02, help="a brightness contour is drawn only where the light changes at least this much per pixel along it")
    ap.add_argument("--min-len", type=float, default=36, help="drop lines shorter than this many pixels")
    ap.add_argument("--min-size", type=float, default=10, help="drop shapes smaller than this many pixels across")
    ap.add_argument("--limit", type=int, default=0, help="stop after this many frames")
    a = ap.parse_args()

    centres, skin_lab, total, n_cuts = make_palette(a.video, a.width, a.fps, a.colours, a.merge)
    shot_mode = n_cuts >= 2  # a montage: every shot gets its own palette and starts with no memory
    if n_cuts:
        print(f"{n_cuts} cuts found", file=sys.stderr)

    def with_skin(c, s):  # skin is the last palette entry, claimed by the colour rule not by nearness
        return (np.vstack([c, s[None, :]]) if s is not None else c), (len(c) if s is not None else -1)

    centres, skin_k = with_skin(centres, skin_lab)
    K = len(centres)
    sink = sys.stdout if a.out == "-" else open(a.out, "w")
    levels = [(k + 1) / (a.tones + 1) for k in range(a.tones)]

    prob = tone = edge = imp = prev_gray = prev_ids = prev_clean = prev_raw = prev_label = None
    next_id = 0
    paint = {}  # the colour each tracked patch is painted
    src = read_frames(a.video, a.width, a.fps)
    ahead = []
    last_small = None
    i = -1
    while True:
        while len(ahead) < 8:  # look a few frames ahead, to see where this shot ends
            try:
                ahead.append(next(src))
            except StopIteration:
                break
        if not ahead:
            break
        frame = ahead.pop(0)
        i += 1
        if a.limit and i >= a.limit:
            break
        h, w = frame.shape[:2]
        small = tiny(frame)
        if last_small is None or (shot_mode and is_cut(last_small, small)):
            if last_small is not None:  # a cut: forget everything about the last shot
                prob = tone = edge = imp = prev_gray = prev_ids = prev_clean = prev_raw = prev_label = None
                paint = {}
            if shot_mode:
                shot = [frame]
                prev_t = small
                for nxt in ahead:
                    t = tiny(nxt)
                    if is_cut(prev_t, t):
                        break
                    shot.append(nxt)
                    prev_t = t
                c, s = palette_from(shot, a.colours, a.merge)
                centres, skin_k = with_skin(c, s)
                K = len(centres)
        last_small = small
        raw_gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
        flow = steady(flow_between(raw_gray, prev_raw)) if prev_raw is not None else None
        imp = importance(flow, imp) if flow is not None else np.zeros((h, w), np.float32)
        if flow is not None:
            # wherever the new frame still disagrees with the last one carried forward, the flow was wrong
            # (a swinging arm, something new appearing) and the history there is a ghost: forget it
            gap = cv2.GaussianBlur(np.abs(raw_gray.astype(np.float32) - warp(prev_raw, flow).astype(np.float32)), (0, 0), 3)
            imp = np.maximum(imp, np.clip((gap - 4.0) / 14.0, 0.0, 1.0).astype(np.float32))
        keep = memory(a.calm, imp)
        clean = denoise(frame, prev_clean, flow)  # from here on, the footage with its noise taken out
        prev_clean, prev_raw = clean, raw_gray
        frame = lift(np.clip(clean, 0, 255).astype(np.uint8))  # and brightened if it is very dark
        gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
        tex = texture(gray)

        # --- labels, voted on by last frame's moved along the flow
        flat = flatten(frame, 8, a.flat)
        lab_img = to_lab(flat)
        lbl = label_frame(lab_img, centres[:skin_k] if skin_k >= 0 else centres)
        if skin_k >= 0:
            lbl[skin_mask(flat) > 0] = skin_k
        now = np.zeros((h, w, K), np.float32)
        np.put_along_axis(now, lbl[..., None], 1.0, axis=2)
        prob = now if prob is None else keep[..., None] * warp(prob, flow) + (1 - keep[..., None]) * now
        label = cv2.medianBlur(hold(prob, prev_label, flow, a.hold, imp), 5)
        fine_label = label.copy()  # the same picture before merging: source of the small detail patches
        label = merge_weak(label, centres, gray, K, a.border, a.merge_colour, lab_img)
        label = absorb_small(label, K, a.min_area, imp, skin_k, centres, a.stand_out)
        prev_label = label.copy()  # the settled picture, merges and all, is what the next frame holds on to

        # --- identities of the patches
        ids, owner, next_id = give_ids(label, flow, prev_ids, next_id)
        prev_ids = owner[ids]

        # --- each patch is painted the footage's own colour (its mean over the whole patch), kept steady
        # over time per identity, so a patch whose footage does not change never changes colour
        seen = set()
        regions = []
        for k in range(K):
            for area, loops, contrast, now_rgb, pid in shapes(label, k, centres, a.min_area, a.smooth, frame, prev_ids):
                now_c = np.array(now_rgb)
                old_c = paint.get(pid) if pid else None
                if old_c is None:
                    col = now_c
                else:  # drift slowly, but follow a real change quickly
                    rate = 0.6 if float(np.abs(now_c - old_c).max()) > 40 else 0.12
                    col = old_c * (1 - rate) + now_c * rate
                if pid:
                    paint[pid] = col
                    seen.add(pid)
                px, py = loops[0][0]
                regions.append({"c": [int(round(v)) for v in col], "area": area, "x": round(contrast, 1), "id": pid, "loops": loops,
                                "_c": (min(max(int(px), 0), w - 1), min(max(int(py), 0), h - 1)), "_sat": float(max(col) - min(col))})
        for pid in [q for q in paint if q not in seen]:
            del paint[pid]  # forget patches that are gone
        regions.sort(key=lambda r: -r["area"])

        # --- lines from brightness and from edge strength, each smoothed over time
        gl = lift_shadows(gray).astype(np.float32) / 255.0
        coarse = cv2.GaussianBlur(gl, (0, 0), a.blur * 1.2)
        fine = cv2.GaussianBlur(gl, (0, 0), max(0.6, a.blur * 0.4))
        # busy places (leaves, gravel) have many small edges packed together; blur those places much harder so a
        # whole canopy becomes one shape, not a hundred leaf outlines. Moving things keep their detail.
        ed = np.clip(cv2.Canny(cv2.GaussianBlur(gray, (0, 0), 1.2), 40, 110).astype(np.float32) / 255.0, 0, 1)
        dens_e = cv2.GaussianBlur(ed, (0, 0), 10)
        cl = float(np.percentile(dens_e, a.clutter))
        busy_map = np.clip((dens_e - cl) / max(cl, 1e-4), 0.0, 1.0) * (1.0 - imp)
        busy_map = cv2.GaussianBlur(busy_map, (0, 0), 6)
        flatter = cv2.GaussianBlur(gl, (0, 0), a.blur * 5.0)
        t_now = imp * fine + (1 - imp) * ((1 - busy_map) * coarse + busy_map * flatter)
        tone = t_now if tone is None else keep * warp(tone, flow) + (1 - keep) * t_now
        sm = cv2.GaussianBlur(gl, (0, 0), 1.0)
        sx, sy = cv2.Sobel(sm, cv2.CV_32F, 1, 0), cv2.Sobel(sm, cv2.CV_32F, 0, 1)
        e_now = np.sqrt(np.clip(np.hypot(sx, sy) / 1.6, 0, 1))  # a fixed scale, so the threshold means the same in every frame
        edge = e_now if edge is None else keep * warp(edge, flow) + (1 - keep) * e_now
        if a.style == "pencil":
            pen = Pencil(h, w, imp, tex, a)
            pen.silhouettes(regions)
            pen.layer(contour_lines(edge, a.edge_cut, imp, tex, a, 1, w, h), 1, lambda l: 1.3)
            tg = np.hypot(cv2.Sobel(tone, cv2.CV_32F, 1, 0), cv2.Sobel(tone, cv2.CV_32F, 0, 1)) / 8.0
            for lv in levels:
                pen.layer(contour_lines(tone, lv, imp, tex, a, 2, w, h, tg), 2, lambda l: 0.7 + 0.7 * (1.0 - l["l"]))
            lines = pen.lines
        else:  # contour: thin lines straight from the footage's own light and edges, no colour patches involved
            lines = []
            for lv in levels:
                for l in contour_lines(tone, lv, imp, tex, a, 2, w, h):
                    l["w"] = round(1.8 * (0.45 + 0.9 * lv), 2)
                    lines.append(l)
            for l in contour_lines(edge, a.edge_cut, imp, tex, a, 1, w, h):
                l["w"] = 1.8
                lines.append(l)
            # clutter: leaves and gravel make many short lines packed tightly together. Where the lines are
            # densest, keep only the long ones (those are real shapes); moving things are always kept.
            ras = np.zeros((h, w), np.float32)
            for l in lines:
                cv2.polylines(ras, [np.round(np.array(l["p"])).astype(np.int32)], False, 1.0, 1)
            dens = cv2.GaussianBlur(ras, (0, 0), 14)
            cut = float(np.percentile(dens[dens > 0], a.dense)) if (dens > 0).any() else 1e9
            kept = []
            for l in lines:
                p = np.array(l["p"])
                length = float(np.hypot(*np.diff(p, axis=0).T).sum()) if len(p) > 1 else 0.0
                mx, my = int(np.clip(p[:, 0].mean(), 0, w - 1)), int(np.clip(p[:, 1].mean(), 0, h - 1))
                if dens[my, mx] > cut and length < a.min_len * 4 and imp[my, mx] < 0.3:
                    continue
                kept.append(l)
            lines = kept
            for r in regions:  # a small thing that differs a lot from what is around it (a cone, a lamp) gets its outline
                if r["x"] >= a.stand_out and r["area"] < 5000:
                    for loop in r["loops"][:1]:
                        pts = np.array(loop + [loop[0]], np.float32) + 0.5
                        if len(pts) >= 4:
                            lines.append({"k": 0, "l": 0.0, "w": 1.6, "p": np.round(pts, 2).tolist()})
        for r in regions:
            del r["_c"], r["_sat"]

        # --- detail: small patches that differ clearly from the big patch under them go on top, so a sun
        # does not melt into its sky; busy texture (leaves, sparks) and anything faint is left out
        if a.detail_max > 0:
            fine_label = absorb_small(fine_label, K, a.detail_area, imp, skin_k, centres, a.stand_out)
            n_comp = int(ids.max()) + 1
            counts = np.maximum(np.bincount(ids.ravel(), minlength=n_comp), 1)
            base_rgb = np.stack([np.bincount(ids.ravel(), weights=frame[..., 2 - ch].ravel(), minlength=n_comp) / counts for ch in range(3)], axis=1)
            base_img = base_rgb[ids]
            zeros = np.zeros((h, w), np.int32)
            extra = []
            for k in range(K):
                for area, loops, contrast, now_rgb, _ in shapes(fine_label, k, centres, a.detail_area, a.smooth, frame, zeros):
                    px, py = min(max(int(loops[0][0][0]), 0), w - 1), min(max(int(loops[0][0][1]), 0), h - 1)
                    if float(np.linalg.norm(np.array(now_rgb) - base_img[py, px])) < a.detail_min:
                        continue
                    if tex[py, px] > 0.6 and imp[py, px] < 0.3 and contrast < a.stand_out:
                        continue
                    extra.append({"c": [int(round(v)) for v in now_rgb], "area": area, "x": round(contrast, 1), "id": 0, "loops": loops})
            extra.sort(key=lambda r: -r["area"])
            regions += extra[: a.detail_max]
        sink.write(json.dumps({"i": i, "w": w, "h": h, "regions": regions, "lines": lines}, separators=(",", ":")) + "\n")
        sink.flush()
        prev_gray = gray
        if i % 20 == 0:
            print(f"frame {i}/{total}", file=sys.stderr)
    if sink is not sys.stdout:
        sink.close()


if __name__ == "__main__":
    main()
