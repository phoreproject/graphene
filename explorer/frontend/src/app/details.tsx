"use client";

import Code from "@/components/Code";
import { GetBlocksResponse } from "@/resources/types";
import { getBlocks } from "@/resources/useBlocks";
import { formatHash } from "@/utils/formatHash";
import Link from "next/link";
import React from "react";
import useSWR from "swr";

const HomeDetails = ({
  initialResponse,
}: {
  initialResponse: GetBlocksResponse;
}) => {
  const { data: blocksResponse } = useSWR(["blocks"], getBlocks, {
    fallbackData: initialResponse,
    refreshInterval: 2000,
  });

  return (
    <>
      <h1 className="text-2xl font-semibold pb-4 pt-2">Blocks</h1>
      <div className="w-full">
        <div className="flex grid-cols-2 font-bold">
          <div className="w-48">Time</div>
          <div className="w-24">Slot</div>
          <div className="flex-grow">Hash</div>
          <div>Proposer</div>
        </div>
        {blocksResponse.Blocks.map((block) => (
          <React.Fragment key={block.BlockHash}>
            <div className="flex">
              <Link
                href={`/block/${block.BlockHash}`}
                className="flex-grow flex"
              >
                <div className="w-48">
                  <p>{new Date(block.Timestamp * 1000).toLocaleString()}</p>
                </div>
                <div className="w-24">
                  <p>{block.Slot}</p>
                </div>
                <div className="flex-grow">
                  <Code includeHash>{formatHash(block.BlockHash)}</Code>
                </div>
              </Link>
              <Link href={`/validator/${block.ProposerHash}`}>
                <p>
                  <Code includeHash>{formatHash(block.ProposerHash)}</Code>
                </p>
              </Link>
            </div>
          </React.Fragment>
        ))}
      </div>
      <h1 className="text-2xl font-semibold pb-4 pt-6">Transactions</h1>
    </>
  );
};

export default HomeDetails;
