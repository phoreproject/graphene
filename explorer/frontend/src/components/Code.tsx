import clsx from "clsx";

const Code = ({
  children,
  includeHash = false,
  className,
}: {
  children: string;
  includeHash?: boolean;
  className?: string;
}): JSX.Element => {
  const blockColor = includeHash ? `#${children.slice(0, 6)}` : null;

  return (
    <pre
      className={clsx(
        "text-sm bg-gray-100 inline px-1 py-0.5 rounded",
        className
      )}
    >
      {blockColor && (
        <div
          className="text-xs inline-block w-2.5 h-2.5 rounded mr-2"
          style={{ backgroundColor: blockColor }}
        ></div>
      )}
      {children}
    </pre>
  );
};

export default Code;
