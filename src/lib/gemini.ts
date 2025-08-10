import { GoogleGenerativeAI } from "@google/generative-ai";
import { get } from "svelte/store";
import { settings } from "./settings";

export class GeminiService {
  private static instance: GeminiService;
  private genAI: GoogleGenerativeAI | null = null;
  private lastApiKey: string = "";
  
  private constructor() {}
  
  static getInstance(): GeminiService {
    if (!GeminiService.instance) {
      GeminiService.instance = new GeminiService();
    }
    return GeminiService.instance;
  }
  
  private getOrInitializeClient(): GoogleGenerativeAI {
    const $settings = get(settings);
    
    if (!$settings.geminiApiKey) {
      throw new Error("Gemini API key is not configured");
    }
    
    // Reinitialize if API key changed
    if (!this.genAI || this.lastApiKey !== $settings.geminiApiKey) {
      this.genAI = new GoogleGenerativeAI($settings.geminiApiKey);
      this.lastApiKey = $settings.geminiApiKey;
    }
    
    return this.genAI;
  }
  
  async queryGemini(prompt: string, context?: string): Promise<string> {
    const $settings = get(settings);
    
    if (!$settings.aiEnabled) {
      throw new Error("AI integration is disabled");
    }
    
    if (!$settings.geminiApiKey) {
      throw new Error("Please configure your Gemini API key in settings");
    }
    
    try {
      const genAI = this.getOrInitializeClient();
      
      // Get the model
      const modelName = $settings.aiModel || "gemini-2.5-flash";
      console.log("Using model:", modelName);
      
      const model = genAI.getGenerativeModel({ 
        model: modelName,
        generationConfig: {
          temperature: 0.7,
          topK: 40,
          topP: 0.95,
          maxOutputTokens: $settings.aiMaxResponseTokens || 4096,
        },
      });
      
      // Build the full prompt with context
      let fullPrompt = prompt;
      if (context) {
        fullPrompt = `Context (terminal output):\n\`\`\`\n${context}\n\`\`\`\n\nUser query: ${prompt}`;
      }
      
      console.log("📤 Sending to Gemini API:", {
        model: modelName,
        promptLength: fullPrompt.length,
        hasContext: !!context,
        contextLength: context?.length || 0,
        fullPrompt: fullPrompt,
      });
      
      // Generate content
      const result = await model.generateContent(fullPrompt);
      
      if (!result.response) {
        throw new Error("No response received from Gemini");
      }
      
      const text = result.response.text();
      
      if (!text) {
        console.error("Empty response from Gemini:", result);
        throw new Error("Empty response received from Gemini");
      }
      
      console.log("📥 Received from Gemini:", {
        responseLength: text.length,
        responsePreview: text.substring(0, 300) + (text.length > 300 ? '...' : ''),
      });
      return text;
      
    } catch (error) {
      console.error("Gemini API error:", error);
      
      if (error instanceof Error) {
        // Handle specific API errors
        if (error.message.includes("API_KEY_INVALID") || error.message.includes("401")) {
          throw new Error("Invalid API key. Please check your Gemini API key in settings.");
        } else if (error.message.includes("QUOTA_EXCEEDED") || error.message.includes("429")) {
          throw new Error("API quota exceeded. Please try again later.");
        } else if (error.message.includes("MODEL_NOT_FOUND") || error.message.includes("404")) {
          throw new Error(`Model "${$settings.aiModel}" not available. Please choose a different model in settings.`);
        } else if (error.message.includes("400")) {
          throw new Error("Invalid request. The selected model may not be available with your API key.");
        }
        throw error;
      }
      throw new Error("Failed to query Gemini API");
    }
  }
  
  async explainTerminalOutput(selectedText: string): Promise<string> {
    const prompt = `Please explain this terminal output in a clear and concise way. If it's an error, suggest how to fix it. If it's command output, explain what it means. Be practical and actionable.`;
    return this.queryGemini(prompt, selectedText);
  }
  
  async askCustom(question: string, selectedText: string): Promise<string> {
    return this.queryGemini(question, selectedText);
  }
}

export const geminiService = GeminiService.getInstance();