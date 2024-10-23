# -*- coding: utf-8 -*-

import cv2
import numpy as np


def is_rectangle(contour):
    # 使用多边形逼近
    epsilon = 0.1 * cv2.arcLength(contour, True)
    approx = cv2.approxPolyDP(contour, epsilon, True)

    # 判断是否有4个顶点
    if len(approx) == 4:
        # 检查是否是凸形
        if cv2.isContourConvex(approx):
            # 检查每个角度是否接近90度
            for i in range(4):
                pt1 = approx[i][0]
                pt2 = approx[(i + 1) % 4][0]
                pt3 = approx[(i + 2) % 4][0]
                
                # 向量 (pt1 -> pt2) 和 (pt2 -> pt3)
                v1 = pt2 - pt1
                v2 = pt3 - pt2

                # 计算向量之间的夹角
                angle = np.degrees(np.arccos(np.dot(v1, v2) / (np.linalg.norm(v1) * np.linalg.norm(v2))))
                
                # 检查角度是否接近 90 度 (可以允许一定的误差，如±10度)
                if not (80 <= angle <= 100):
                    return False

            return True
    return False


if __name__ == '__main__':
    orig = cv2.imread("poker.png")
    _, thresh = cv2.threshold(cv2.cvtColor(orig, cv2.COLOR_BGR2GRAY), 150, 255, cv2.THRESH_BINARY)
    contours, _ = cv2.findContours(image=thresh, mode=cv2.RETR_TREE, method=cv2.CHAIN_APPROX_NONE)

    def filter_contour(contour):
        _, _, w, h = cv2.boundingRect(contour)
        if w*h > 200 and h/w > 1.2 and h/w < 1.5:
            return is_rectangle(contour)
        return False

    contours = list(filter(filter_contour, contours))
    cv2.drawContours(orig, contours, -1, (0, 0, 255), 3)

    cv2.imwrite("contour.png", orig)
    cv2.imshow("contours", orig)
    cv2.waitKey(0)
    cv2.destroyAllWindows()
