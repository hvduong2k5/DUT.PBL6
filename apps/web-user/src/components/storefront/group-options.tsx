import Image from "next/image";
import { collections } from "@/components/layout/storefront-links";

const subtitles = [
  "Dẻo mạch nha · Giòn mè rang",
  "Trà sen Tịnh Tâm · Trà cung đình",
  "Hộp giấy Dó · Nếp gấp 45°",
];
const descriptions = [
  "Thức quà ngọt lành từ mạch nha, đậu phộng và mè rang vàng óng.",
  "Hương sen hồ Tịnh Tâm và thảo mộc cung đình thanh tao, dịu ngọt.",
  "Hộp quà bồi giấy Dó trang nhã, gói trọn tấm lòng người trao.",
];

export function GroupOptions({
  choose,
  menu = false,
  selected,
}: {
  choose: (index: number) => void;
  menu?: boolean;
  selected?: number;
}) {
  return collections.map((group, i) =>
    menu ? (
      <button
        key={group.href}
        type="button"
        className="mega-cat-item"
        onClick={() => choose(i)}
      >
        <Image
          src={"/images/goi-hue/" + group.image}
          alt=""
          width={400}
          height={240}
        />
        <strong>{group.name}</strong>
        <span>{descriptions[i]}</span>
        <small>Khám phá nhóm món →</small>
      </button>
    ) : (
      <button
        key={group.href}
        type="button"
        className={"cat-card" + (selected === i ? " is-active" : "")}
        onClick={() => choose(i)}
      >
        <div className="cat-card__media">
          <Image
            src={"/images/goi-hue/" + group.image}
            alt=""
            width={540}
            height={320}
          />
        </div>
        <div className="cat-card__foot">
          <div>
            <span className="cat-card__title">{group.name}</span>
            <span className="cat-card__sub">{subtitles[i]}</span>
          </div>
          <span className="cat-card__arrow" aria-hidden="true">
            →
          </span>
        </div>
      </button>
    ),
  );
}
