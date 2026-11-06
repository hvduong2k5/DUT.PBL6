import type { Metadata } from "next";
import { ProfileFlow } from "./profile-flow";

export const metadata: Metadata = { title: "Hồ sơ cá nhân" };

export default function ProfilePage() {
  return <ProfileFlow />;
}
