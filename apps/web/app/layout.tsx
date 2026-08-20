export const metadata = {
  title: "CompanyOS",
  description: "Organizational runtime for semi-autonomous AI companies.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
