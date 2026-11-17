import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { HomeExperience } from "./home-experience";

export const metadata = { title: { absolute: "Gói Huế · Phong vị Huế" }, description: "Mè xửng, trà và quà biếu từ Huế." };

export default function HomePage() {
  return (
    <><SiteHeader /><HomeExperience /><SiteFooter /></>
  );
}
