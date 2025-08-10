import { get } from "svelte/store";
import { settings, MODEL_CONTEXT_WINDOWS } from "./settings";
import { geminiService } from "./gemini";
import { openRouterService } from "./openrouter";

// Simple token estimation: ~4 characters per token on average
// This is a rough approximation - actual tokenization varies by model
export function estimateTokens(text: string): number {
  return Math.ceil(text.length / 4);
}

export interface ConversationEntry {
  role: string;
  content: string;
}

export class ContextManager {
  private static instance: ContextManager;
  
  private constructor() {}
  
  static getInstance(): ContextManager {
    if (!ContextManager.instance) {
      ContextManager.instance = new ContextManager();
    }
    return ContextManager.instance;
  }
  
  /**
   * Get the effective context window for the current model
   */
  getContextWindow(): number {
    const $settings = get(settings);
    
    // If user has set a custom context length, use that
    if ($settings.aiContextLength && $settings.aiContextLength > 0) {
      return $settings.aiContextLength;
    }
    
    // Otherwise use model default
    const currentModel = $settings.aiProvider === 'gemini' 
      ? $settings.aiModel 
      : $settings.openRouterModel;
      
    return MODEL_CONTEXT_WINDOWS[currentModel] || MODEL_CONTEXT_WINDOWS["default"];
  }
  
  /**
   * Calculate current conversation size in tokens
   */
  calculateConversationTokens(conversation: ConversationEntry[]): number {
    let totalTokens = 0;
    
    for (const entry of conversation) {
      // Add role tokens (estimated)
      totalTokens += estimateTokens(entry.role + ": ");
      // Add content tokens
      totalTokens += estimateTokens(entry.content);
      // Add separator tokens
      totalTokens += 2; // For newlines
    }
    
    return totalTokens;
  }
  
  /**
   * Check if conversation is approaching context limit
   */
  isNearingLimit(conversation: ConversationEntry[], threshold: number = 0.9): boolean {
    const contextWindow = this.getContextWindow();
    const currentTokens = this.calculateConversationTokens(conversation);
    
    return currentTokens >= (contextWindow * threshold);
  }
  
  /**
   * Compress conversation history using AI
   */
  async compressConversation(conversation: ConversationEntry[]): Promise<ConversationEntry[]> {
    const $settings = get(settings);
    
    // Don't compress if there's not much to compress
    if (conversation.length < 4) {
      return conversation;
    }
    
    // Separate context from conversation
    const contextEntry = conversation.find(entry => entry.role === 'Context');
    const conversationOnly = conversation.filter(entry => entry.role !== 'Context');
    
    // Keep the most recent exchange intact
    const recentExchanges = conversationOnly.slice(-2); // Last Q&A pair
    const toCompress = conversationOnly.slice(0, -2); // Everything else
    
    if (toCompress.length < 2) {
      return conversation; // Not enough to compress
    }
    
    // Build compression prompt
    const compressionPrompt = `You are a conversation summarizer. Your task is to compress the following conversation history while preserving ALL critical information, technical details, and context.

CONVERSATION TO COMPRESS:
${toCompress.map(entry => `${entry.role}: ${entry.content}`).join('\n\n')}

INSTRUCTIONS:
1. Create a concise summary that preserves:
   - All technical commands, code, and error messages
   - Key questions asked and solutions provided
   - Important context and decisions made
   - The logical flow and relationship between topics

2. Format as a single comprehensive summary paragraph that can replace the above conversation.

3. Be accurate and complete - losing important technical details is unacceptable.

4. Start your response with "Previous conversation summary:" followed by the compressed content.

Compress the conversation now:`;

    try {
      console.log('🗜️ Compressing conversation:', {
        originalEntries: toCompress.length,
        tokensBeforeCompression: this.calculateConversationTokens(toCompress),
      });
      
      // Use the appropriate AI service
      const compressedText = $settings.aiProvider === 'openrouter' 
        ? await openRouterService.queryOpenRouter(compressionPrompt)
        : await geminiService.queryGemini(compressionPrompt);
      
      // Create new compressed conversation
      const compressedConversation: ConversationEntry[] = [];
      
      // Add context if it exists
      if (contextEntry) {
        compressedConversation.push(contextEntry);
      }
      
      // Add compressed summary as a system message
      compressedConversation.push({
        role: 'System',
        content: compressedText
      });
      
      // Add recent exchanges
      compressedConversation.push(...recentExchanges);
      
      const tokensAfter = this.calculateConversationTokens(compressedConversation);
      const tokensBefore = this.calculateConversationTokens(conversation);
      
      console.log('✅ Compression complete:', {
        tokensBefore,
        tokensAfter,
        reduction: Math.round((1 - tokensAfter / tokensBefore) * 100) + '%',
      });
      
      return compressedConversation;
      
    } catch (error) {
      console.error('❌ Compression failed:', error);
      // If compression fails, return original conversation
      return conversation;
    }
  }
  
  /**
   * Check and compress if needed
   */
  async checkAndCompress(conversation: ConversationEntry[]): Promise<ConversationEntry[]> {
    const $settings = get(settings);
    
    // Only compress if auto-compress is enabled
    if (!$settings.aiAutoCompress) {
      return conversation;
    }
    
    // Check if we're nearing the limit
    if (this.isNearingLimit(conversation, 0.9)) {
      console.log('⚠️ Approaching context limit, initiating compression...');
      return await this.compressConversation(conversation);
    }
    
    return conversation;
  }
  
  /**
   * Get a status report on current context usage
   */
  getContextStatus(conversation: ConversationEntry[]): {
    currentTokens: number;
    maxTokens: number;
    percentageUsed: number;
    shouldCompress: boolean;
  } {
    const maxTokens = this.getContextWindow();
    const currentTokens = this.calculateConversationTokens(conversation);
    const percentageUsed = (currentTokens / maxTokens) * 100;
    
    return {
      currentTokens,
      maxTokens,
      percentageUsed,
      shouldCompress: percentageUsed >= 90
    };
  }
}

export const contextManager = ContextManager.getInstance();