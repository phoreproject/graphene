import { getBlock } from "@/resources/useBlock";
import BlockDetails from "./details";

const BlockDetailsPage = async ({
  params: { blockID },
}: {
  params: { blockID: string };
}) => {
  const block = await getBlock(blockID);

  return (
    <div>
      <BlockDetails initialResponse={block} />
    </div>
  );
};

export default BlockDetailsPage;
