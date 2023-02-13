import Link from "next/link";
import "./globals.scss";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head />
      <body>
        <main className="max-w-screen-lg mx-auto py-8">
          <h1 className="text-4xl font-bold">Graphene Explorer</h1>
          <div className="flex justify-between pb-4 pt-2">
            <Link href="/">Recent Blocks</Link>
            <Link href="/">Recent Transactions</Link>
            <Link href="/">Shard Blocks</Link>
            <Link href="/">Shard Transactions</Link>
            <Link href="/">Block Votes</Link>
          </div>
          {children}
        </main>
      </body>
    </html>
  );
}
