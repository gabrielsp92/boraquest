// Line icons: 24px grid, 2.5px stroke, round caps. Add new ones in the same style.
import type { ReactNode } from "react";

const paths = {
  check: (<><path d="M5 12.5l4.5 4.5L19 7.5" /></>),
  plus: (<><path d="M12 5v14M5 12h14" /></>),
  minus: (<><path d="M5 12h14" /></>),
  close: (<><path d="M6 6l12 12M18 6L6 18" /></>),
  camera: (<><path d="M4 8h3l2-3h6l2 3h3v11H4z" /><circle cx={12} cy={13} r={3.5} /></>),
  image: (<><rect x={4} y={5} width={16} height={14} rx={3} /><path d="M4 16l4.5-4.5 4 4 2.5-2.5 5 5" /><circle cx={15.5} cy={9.5} r={1.2} /></>),
  trophy: (<><path d="M8 4h8v5a4 4 0 0 1-8 0z" /><path d="M8 6H4.5c0 2 1.5 3.5 3.8 3.5M16 6h3.5c0 2-1.500 3.5-3.8 3.5M12 13v4M8 20h8" /></>),
  shield: (<><path d="M12 3l7 3v5c0 5-3 8-7 10-4-2-7-5-7-10V6z" /><path d="M9 12l2.2 2.2L15 10.500" /></>),
  sun: (<><circle cx={12} cy={12} r={4} /><path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.600 5.600l1.400 1.400M17 17l1.400 1.400M18.400 5.600L17 7M7 17l-1.400 1.400" /></>),
  calendar: (<><rect x={4} y={5} width={16} height={15} rx={3} /><path d="M4 10h16M8 3v4M16 3v4" /></>),
  users: (<><circle cx={9} cy={9} r={3.5} /><path d="M3 20c0-3.5 2.700-6 6-6s6 2.500 6 6" /><circle cx={17.5} cy={8} r={2.5} /><path d="M17 14c2.400.300 4 2.500 4 6" /></>),
  scroll: (<><rect x={4} y={3} width={16} height={18} rx={3} /><path d="M8 8h8M8 12h8M8 16h5" /></>),
  crown: (<><path d="M4 18h16l1-10-5 4-4-7-4 7-5-4z" /></>),
  gift: (<><rect x={4} y={11} width={16} height={9} rx={1} /><path d="M3 7h18v4H3zM12 7v13M12 7c-2-4.500-6-3-4.500 0M12 7c2-4.500 6-3 4.500 0" /></>),
  pencil: (<><path d="M4 20l1-4L16 5l3 3L8 19z" /></>),
  chev: (<><path d="M9 6l6 6-6 6" /></>),
} satisfies Record<string, ReactNode>;

export type IconName = keyof typeof paths;

export function Icon({ name, size = 24 }: { name: IconName; size?: number }) {
  return (
    <svg className="bq-icon" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {paths[name]}
    </svg>
  );
}
