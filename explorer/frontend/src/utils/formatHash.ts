export const formatHash = (hash: string) => {
  return `${hash.slice(0, 6)}...${hash.slice(-6)}`;
};
