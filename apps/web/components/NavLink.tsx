import type { CSSProperties, ReactNode } from "react";
import Link from "next/link";

export default function NavLink({
  href,
  active,
  navBtnStyle,
  children,
}: {
  href: string;
  active: boolean;
  navBtnStyle: (active: boolean) => CSSProperties;
  children: ReactNode;
}) {
  return (
    <Link href={href} className="navbtn" aria-current={active ? "page" : undefined} style={navBtnStyle(active)}>
      {children}
    </Link>
  );
}
