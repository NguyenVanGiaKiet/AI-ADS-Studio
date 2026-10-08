import json
import sys

import cv2


def main():
    if len(sys.argv) != 2:
        raise ValueError("usage: detect_faces.py <video-path>")

    video_path = sys.argv[1]
    capture = cv2.VideoCapture(video_path)
    if not capture.isOpened():
        raise RuntimeError(f"không thể mở video để nhận diện khuôn mặt: {video_path}")

    duration = capture.get(cv2.CAP_PROP_FRAME_COUNT) / capture.get(cv2.CAP_PROP_FPS)
    if duration <= 0:
        capture.release()
        raise RuntimeError(f"video không có thời lượng hợp lệ: {video_path}")

    cascade_path = cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
    detector = cv2.CascadeClassifier(cascade_path)
    if detector.empty():
        capture.release()
        raise RuntimeError(f"không tải được bộ nhận diện khuôn mặt: {cascade_path}")

    sample_step = 0.5
    exclusion_margin = 0.65
    face_intervals = []
    timestamp = 0.0
    while timestamp < duration:
        capture.set(cv2.CAP_PROP_POS_MSEC, timestamp * 1000)
        ok, frame = capture.read()
        if ok:
            height, width = frame.shape[:2]
            if width > 640:
                resized_height = max(1, round(height * 640 / width))
                frame = cv2.resize(frame, (640, resized_height))
            gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            faces = detector.detectMultiScale(
                gray,
                scaleFactor=1.1,
                minNeighbors=5,
                minSize=(28, 28),
            )
            if len(faces):
                start = max(0.0, timestamp - exclusion_margin)
                end = min(duration, timestamp + exclusion_margin)
                face_intervals.append((start, end))
        timestamp += sample_step
    capture.release()

    merged = []
    for start, end in face_intervals:
        if merged and start <= merged[-1][1]:
            merged[-1] = (merged[-1][0], max(merged[-1][1], end))
        else:
            merged.append((start, end))

    safe_segments = []
    cursor = 0.0
    for start, end in merged:
        if start - cursor >= 1.0:
            safe_segments.append({"start": cursor, "duration": start - cursor})
        cursor = max(cursor, end)
    if duration - cursor >= 1.0:
        safe_segments.append({"start": cursor, "duration": duration - cursor})

    print(json.dumps({"duration": duration, "safeSegments": safe_segments}))


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
