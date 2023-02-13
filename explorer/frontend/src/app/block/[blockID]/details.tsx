"use client";

import Code from "@/components/Code";
import { Block } from "@/resources/types";
import { getBlock } from "@/resources/useBlock";
import { formatHash } from "@/utils/formatHash";
import Link from "next/link";
import React from "react";
import useSWR from "swr";

const BlockDetails = ({
  initialResponse,
}: {
  initialResponse: Block;
}): JSX.Element => {
  const { data } = useSWR(
    ["block", initialResponse.BlockHash],
    async ([_k, blockHash]) => await getBlock(blockHash),
    {
      fallbackData: initialResponse,
    }
  );

  console.log(data.BlockHash);

  return (
    <div className="space-y-2">
      <div className="font-bold">Block Details</div>
      <div>
        Block Hash: <Code includeHash>{formatHash(data.BlockHash)}</Code>
      </div>
      <div>Timestamp: {new Date(data.Timestamp * 1000).toLocaleString()}</div>
      <div>Height: {data.Height}</div>
      <div>Slot: {data.Slot}</div>
      <div>
        Parent:{" "}
        <Link href={`/block/${data.ParentBlock}`}>
          <Code includeHash>{formatHash(data.ParentBlock)}</Code>
        </Link>
      </div>
      <div>
        Proposer:{" "}
        <Link href={`/validator/${data.ProposerHash}`}>
          <Code includeHash>{formatHash(data.ProposerHash)}</Code>
        </Link>
      </div>
      <div>
        State: <Code includeHash>{formatHash(data.StateRoot)}</Code>
      </div>
    </div>
  );
};

export default BlockDetails;
