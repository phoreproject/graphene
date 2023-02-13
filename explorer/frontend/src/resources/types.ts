export interface Block {
  ProposerHash: string;
  ParentBlock: string;
  StateRoot: string;
  RandaoReveal: string;
  Signature: string;
  BlockHash: string;
  BlockHeight: number;
  Slot: number;
  Timestamp: number;
}

export interface Transaction {
  Amount: number;
  RecipientHash: string;
  Type: number;
  Slot: number;
}

export interface GetBlocksResponse {
  Blocks: Block[];
  Transactions: Transaction[];
}
