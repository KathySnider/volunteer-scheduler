import "./globals.css";

export const metadata = {
  title: "Volunteer Scheduler System",
  description: "Volunteer event scheduling and management",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
