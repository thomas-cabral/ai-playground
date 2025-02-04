export interface Message {
  id: number;
  chatId: number;
  role: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
  modelName: string;
  starred: boolean;
  tokenUsage?: TokenUsage;
}

export interface Chat {
  id: number;
  messages: Message[];
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
  modelName: string;
  starred: boolean;
  parentId?: number | null;
  forkMessageID?: number;
  tokenUsage?: TokenUsage;
}

type NewMessage = {
  role: string;
  content: string;
  id?: number;
  modelName?: string;
  starred?: boolean;
  tokenUsage?: TokenUsage;
};

type ModelPricing = {
  prompt: string;
  completion: string;
  image: string;
  request: string;
};

type ModelArchitecture = {
  modality: string;
  tokenizer: string;
  instruct_type: string | null;
};

type OpenRouterModel = {
  id: string;
  name: string;
  description?: string;
  pricing: ModelPricing;
  architecture: ModelArchitecture;
};

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

export type { Message, NewMessage, Chat, ModelPricing, ModelArchitecture, OpenRouterModel };