/* eslint-disable @next/next/no-img-element */
// Boraquest components. Each wraps the bq-* classes defined in app/globals.css.
// Usage rules for every one of them are in design/components/<Name>/README.md.
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Icon, type IconName } from "./Icon";
import type { Member } from "@/lib/data";

const cx = (...c: (string | false | undefined)[]) => c.filter(Boolean).join(" ");

/* ---------- Button ---------- */
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "plain" | "ink" | "ghost" | "small";
  block?: boolean;
  icon?: IconName;
};
export function Button({ variant = "primary", block, icon, className, children, ...rest }: ButtonProps) {
  return (
    <button
      type="button"
      className={cx("bq-btn", variant !== "primary" && `bq-btn--${variant}`, block && "bq-btn--block", className)}
      {...rest}
    >
      {icon && <Icon name={icon} size={variant === "small" ? 18 : 24} />}
      {children}
    </button>
  );
}
export function IconButton({ icon, label, ...rest }: ButtonHTMLAttributes<HTMLButtonElement> & { icon: IconName; label: string }) {
  return (
    <button type="button" className="bq-iconbtn" aria-label={label} {...rest}>
      <Icon name={icon} />
    </button>
  );
}

/* ---------- Avatar ---------- */
export function Avatar({ member, size, selected, crown }: { member: Member; size?: "sm" | "lg" | "xl"; selected?: boolean; crown?: boolean }) {
  return (
    <span className={cx("bq-avatar", size && `bq-avatar--${size}`, selected && "bq-avatar--selected")}>
      <img src={member.avatar} alt={member.name} />
      {crown && (
        <span className="bq-avatar__crown">
          <Icon name="crown" />
        </span>
      )}
    </span>
  );
}

/* ---------- Points ---------- */
export const formatPoints = (n: number) => `${n > 0 ? "+" : "−"}${Math.abs(n)}`;
export function Points({ value }: { value: number }) {
  return <span className={cx("bq-points", value > 0 ? "bq-points--gain" : "bq-points--loss")}>{formatPoints(value)}</span>;
}

/* ---------- Quest ---------- */
type QuestProps = {
  name: string;
  points: number;
  /** todo/done: Today checklist. gain/loss: Rules list and logged slips. */
  kind: "todo" | "done" | "gain" | "loss";
  meta?: string;
  selected?: boolean;
  onClick?: () => void;
  pressed?: boolean;
};
export function QuestRow({ name, points, kind, meta, selected, onClick, pressed }: QuestProps) {
  const mark: IconName = kind === "gain" ? "plus" : kind === "loss" ? "minus" : "check";
  const className = cx("bq-quest", kind !== "todo" && `bq-quest--${kind}`, selected && "bq-quest--selected");
  const body = (
    <>
      <span className="bq-quest__mark">
        <Icon name={mark} size={22} />
      </span>
      <span className="bq-grow">
        <span className="bq-quest__name block">{name}</span>
        {meta && <span className="bq-caption block">{meta}</span>}
      </span>
      <Points value={points} />
    </>
  );
  return onClick ? (
    <button type="button" className={className} onClick={onClick} aria-pressed={pressed}>
      {body}
    </button>
  ) : (
    <div className={className}>{body}</div>
  );
}

/* ---------- Badge ---------- */
export function Badge({ kind, icon, children }: { kind?: "audit" | "leader" | "gain" | "loss"; icon?: IconName; children: ReactNode }) {
  return (
    <span className={cx("bq-badge", kind && `bq-badge--${kind}`)}>
      {icon && <Icon name={icon} size={14} />}
      {children}
    </span>
  );
}

/* ---------- Score header ---------- */
export function ScoreHeader({ score, label, done, total }: { score: number; label: string; done?: number; total?: number }) {
  return (
    <div className="bq-score">
      <div>
        <p className="bq-display">{score}</p>
        <p className="bq-label">{label}</p>
      </div>
      {total !== undefined && done !== undefined && (
        <div>
          {total <= 8 && (
            <div className="bq-dots" aria-hidden="true">
              {Array.from({ length: total }, (_, i) => (
                <i key={i} className={i < done ? "is-on" : undefined} />
              ))}
            </div>
          )}
          <p className="bq-caption">
            {done} de {total} quests
          </p>
        </div>
      )}
    </div>
  );
}

/* ---------- Top bar ---------- */
export function TopBar({ title, caption, right }: { title: string; caption?: string; right?: ReactNode }) {
  return (
    <header className="bq-topbar">
      <div>
        {caption && <p className="bq-caption">{caption}</p>}
        <h1 className="bq-title">{title}</h1>
      </div>
      {right}
    </header>
  );
}

/* ---------- Sheet ---------- */
export function Sheet({ open, onClose, children }: { open: boolean; onClose: () => void; children: ReactNode }) {
  if (!open) return null;
  return (
    <>
      <button type="button" className="bq-scrim" aria-label="Fechar" onClick={onClose} />
      <div className="bq-sheet" role="dialog" aria-modal="true">
        <div className="bq-sheet__grip" />
        {children}
      </div>
    </>
  );
}

/* ---------- Fields ---------- */
export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="bq-field">
      <span className="bq-label">{label}</span>
      {children}
    </label>
  );
}
export function Segmented<T extends string>({ options, value, onChange }: { options: { value: T; label: string }[]; value: T; onChange: (v: T) => void }) {
  return (
    <div className="bq-seg" role="group">
      {options.map((o) => (
        <button key={o.value} type="button" className="bq-seg__opt" aria-pressed={o.value === value} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  );
}
export function Stepper({ value, onChange, step = 5, min = 5, max = 100, format = String }: { value: number; onChange: (v: number) => void; step?: number; min?: number; max?: number; format?: (v: number) => string }) {
  return (
    <div className="bq-stepper">
      <IconButton icon="minus" label="Menos" onClick={() => onChange(Math.max(min, value - step))} />
      <output>{format(value)}</output>
      <IconButton icon="plus" label="Mais" onClick={() => onChange(Math.min(max, value + step))} />
    </div>
  );
}

/* ---------- Rank row ---------- */
export function RankRow({ position, member, points, leader, crown = leader, children }: { position?: number; member: Member; points: number; leader?: boolean; crown?: boolean; children?: ReactNode }) {
  return (
    <div className={cx("bq-rank", leader && "bq-rank--leader")}>
      {position !== undefined && <span className="bq-rank__pos">{position}</span>}
      <Avatar member={member} crown={crown} />
      <div className="bq-stack bq-grow">
        <p className="bq-text">{member.name}</p>
        {children}
      </div>
      <p className="bq-rank__score">
        {points}
        <small>pontos</small>
      </p>
    </div>
  );
}
export function Bar({ percent }: { percent: number }) {
  return (
    <div className="bq-bar">
      <i style={{ width: `${percent}%` }} />
    </div>
  );
}

/* ---------- Day card ---------- */
export function Thumb({ src }: { src?: string }) {
  return (
    <span className="bq-thumb" role="img" aria-label="Foto de prova">
      {src ? <img src={src} alt="" /> : <Icon name="image" />}
    </span>
  );
}
export function DayCard({ title, gained, lost, photos = 0, children }: { title: string; gained: number; lost: number; photos?: number; children: ReactNode }) {
  return (
    <div className="bq-day">
      <div className="bq-row">
        <h3 className="bq-heading bq-grow">{title}</h3>
        {gained > 0 && <Points value={gained} />}
        {lost < 0 && <Points value={lost} />}
      </div>
      <div className="bq-day__chips">{children}</div>
      {photos > 0 && (
        <div className="bq-thumbs">
          {Array.from({ length: photos }, (_, i) => (
            <Thumb key={i} />
          ))}
        </div>
      )}
    </div>
  );
}

/* ---------- Prize card ---------- */
export function PrizeCard({ label, prize, icon = "trophy" }: { label: string; prize: string; icon?: IconName }) {
  return (
    <div className="bq-prize">
      <span className="bq-prize__icon">
        <Icon name={icon} />
      </span>
      <div>
        <p className="bq-caption">{label}</p>
        <p className="bq-heading">{prize}</p>
      </div>
    </div>
  );
}

/* ---------- Banner ---------- */
export function Banner({ member, title, text }: { member: Member; title: string; text?: string }) {
  return (
    <div className="bq-banner" role="status">
      <Avatar member={member} />
      <div>
        <p className="bq-text">{title}</p>
        {text && <p className="bq-caption">{text}</p>}
      </div>
    </div>
  );
}

/* ---------- Proof ---------- */
export function Proof({ name, day, points, children }: { name: string; day: string; points: number; children: ReactNode }) {
  return (
    <div className="bq-proof">
      <div className="bq-row">
        <div className="bq-grow">
          <p className="bq-text">{name}</p>
          <p className="bq-caption">{day}</p>
        </div>
        <Points value={points} />
      </div>
      {children}
    </div>
  );
}

/* ---------- Empty state ---------- */
export function EmptyState({ icon, title, text, children }: { icon: IconName; title: string; text: string; children?: ReactNode }) {
  return (
    <div className="bq-empty">
      <div className="bq-empty__art" aria-hidden="true">
        <b />
        <b />
        <b />
        <Icon name={icon} size={48} />
      </div>
      <h2 className="bq-title">{title}</h2>
      <p className="bq-text">{text}</p>
      {children}
    </div>
  );
}

/* ---------- Confetti ---------- */
const PIECES = [[6, 4, 20], [18, 12, -30], [30, 3, 45], [44, 9, 10], [58, 2, -20], [70, 11, 35], [84, 5, -45], [93, 14, 15], [10, 24, 60], [26, 30, -15], [76, 27, 25], [90, 33, -35], [4, 44, 10], [95, 50, 40], [14, 58, -25], [86, 62, 30]];
export function Confetti({ fixed }: { fixed?: boolean }) {
  return (
    <div className={cx("bq-confetti", fixed && "bq-confetti--fixed")} aria-hidden="true">
      {PIECES.map(([x, y, r], i) => (
        <i key={i} style={{ left: `${x}%`, top: `${y}%`, rotate: `${r}deg`, animationDelay: `${i * 40}ms` }} />
      ))}
    </div>
  );
}
