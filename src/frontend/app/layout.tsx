import type { Metadata } from "next";
import "./globals.css";
import Navbar from "./components/Navbar";
import { UserAuthWrapper } from "./components/UserWrapper";

export const metadata: Metadata = {
  title: "",
  description: "",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="h-screen flex flex-col">
        <UserAuthWrapper>
          <div
            className="absolute w-full hidden top-0 left-0 p-4 rounded-l text-black"
            id="notification"
          ></div>
          <Navbar />
          <div className="bg-gray-700 flex-grow">{children}</div>
        </UserAuthWrapper>
      </body>
    </html>
  );
}
