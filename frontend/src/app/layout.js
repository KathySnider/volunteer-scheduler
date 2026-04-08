import "./globals.css";

export const metadata = {
  title: "AARP Volunteer System",
  description: "Volunteer event scheduling and management",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
