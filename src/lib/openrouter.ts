import { get } from "svelte/store";
import { settings } from "./settings";

export interface OpenRouterMessage {
  role: "system" | "user" | "assistant";
  content: string;
}

export class OpenRouterService {
  private static instance: OpenRouterService;
  private apiEndpoint = "https://openrouter.ai/api/v1/chat/completions";
  
  private constructor() {}
  
  static getInstance(): OpenRouterService {
    if (!OpenRouterService.instance) {
      OpenRouterService.instance = new OpenRouterService();
    }
    return OpenRouterService.instance;
  }
  
  async queryOpenRouter(prompt: string, context?: string): Promise<string> {
    const $settings = get(settings);
    
    if (!$settings.aiEnabled) {
      throw new Error("AI integration is disabled");
    }
    
    if (!$settings.openRouterApiKey) {
      throw new Error("Please configure your OpenRouter API key in settings");
    }
    
    if (!$settings.openRouterModel) {
      throw new Error("Please select an OpenRouter model in settings");
    }
    
    try {
      // Build messages array for chat completion
      const messages: OpenRouterMessage[] = [];
      
      // Add system message
      messages.push({
        role: "system",
        content: "You are an AI assistant integrated directly into a terminal emulator. The user is viewing your response inline with their terminal session."
      });
      
      // Add context if provided
      if (context) {
        messages.push({
          role: "user",
          content: `Context (terminal output):\n\`\`\`\n${context}\n\`\`\`\n\nUser query: ${prompt}`
        });
      } else {
        messages.push({
          role: "user",
          content: prompt
        });
      }
      
      console.log("📤 Sending to OpenRouter API:", {
        model: $settings.openRouterModel,
        messageCount: messages.length,
        hasContext: !!context,
        contextLength: context?.length || 0,
        fullMessages: messages,
      });
      
      const response = await fetch(this.apiEndpoint, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${$settings.openRouterApiKey}`,
          "Content-Type": "application/json",
          "HTTP-Referer": window.location.origin,
          "X-Title": "SSHXtend Terminal"
        },
        body: JSON.stringify({
          model: $settings.openRouterModel,
          messages: messages,
          temperature: 0.7,
          max_tokens: $settings.aiMaxResponseTokens || 4096,
          top_p: 0.95,
          stream: false
        })
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => null);
        console.error("OpenRouter API error:", response.status, errorData);
        
        if (response.status === 401) {
          throw new Error("Invalid API key. Please check your OpenRouter API key in settings.");
        } else if (response.status === 429) {
          throw new Error("Rate limit exceeded. Please try again later.");
        } else if (response.status === 400) {
          const errorMessage = errorData?.error?.message || "Invalid request";
          throw new Error(`OpenRouter error: ${errorMessage}`);
        } else {
          throw new Error(`OpenRouter API error: ${response.status}`);
        }
      }
      
      const data = await response.json();
      
      if (!data.choices || !data.choices[0]?.message?.content) {
        console.error("Unexpected OpenRouter response:", data);
        throw new Error("No response received from OpenRouter");
      }
      
      const text = data.choices[0].message.content;
      
      console.log("📥 Received from OpenRouter:", {
        responseLength: text.length,
        responsePreview: text.substring(0, 300) + (text.length > 300 ? '...' : ''),
      });
      
      return text;
      
    } catch (error) {
      console.error("OpenRouter API error:", error);
      
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to query OpenRouter API");
    }
  }
  
  async explainTerminalOutput(selectedText: string): Promise<string> {
    const prompt = `Please explain this terminal output in a clear and concise way. If it's an error, suggest how to fix it. If it's command output, explain what it means. Be practical and actionable.`;
    return this.queryOpenRouter(prompt, selectedText);
  }
  
  async askCustom(question: string, selectedText: string): Promise<string> {
    return this.queryOpenRouter(question, selectedText);
  }
}

export const openRouterService = OpenRouterService.getInstance();