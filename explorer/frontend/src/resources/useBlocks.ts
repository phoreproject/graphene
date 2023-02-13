import { Block, GetBlocksResponse, Transaction } from "./types";

import useSWR, { SWRResponse } from "swr";

export const getBlocks = async (): Promise<GetBlocksResponse> => {
  const res = await fetch(process.env.NEXT_PUBLIC_EXPLORER_API_URL);
  // The return value is *not* serialized
  // You can return Date, Map, Set, etc.
  return res.json();
};
