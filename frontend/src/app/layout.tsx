import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Oversite",
  description: "CS2 2D demo viewer and analytics platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full flex flex-col font-sans">{children}</body>
    </html>
  );
}
