// docs/ux/ux-states.md: every empty state names why it's empty and, where
// a real action would resolve it, what to do — never a bare "No data."
export default function Empty({ text }: { text: string }) {
  return <p className="ds-empty">{text}</p>;
}
