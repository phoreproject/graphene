import { getBlocks } from "@/resources/useBlocks";
import HomeDetails from "./details";

export default async function Home() {
  const data = await getBlocks();

  return <HomeDetails initialResponse={data} />;
}
