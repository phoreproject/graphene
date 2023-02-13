import { Block, GetBlocksResponse, Transaction } from "./types";

export const getBlock = async (blockID: string): Promise<Block> => {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_EXPLORER_API_URL}/block/${blockID}`
  );
  // The return value is *not* serialized
  // You can return Date, Map, Set, etc.
  return res.json();
};
