import tkinter as tk
import threading
import time
import json
from tkinter import ttk
from typing import Callable, Optional, Dict, Any, Generator
from transformers import AutoTokenizer
from sdk import RustAIClient

# =====================================================================
# 2. Main Tkinter Application
# =====================================================================
class ChatApp:
    def __init__(self, root):
        self.root = root
        self.chat_history = []
        self.context_size = 40960
        self.tokenizer = AutoTokenizer.from_pretrained("Qwen/Qwen2.5-7B")

        # Initialize SDK pointing to Axum Rust Gateway
        self.client = RustAIClient("http://localhost:8080")

        self.root.title("Krishna's AI Chat - Research Edition (Rust Gateway)")

        # --- Top UI: Controls ---
        self.btn_frame = tk.Frame(root)
        self.btn_frame.pack(pady=5)

        self.use_token_view = tk.BooleanVar()
        self.toggle_switch = ttk.Checkbutton(
            self.btn_frame, 
            text="Enable Tokenizer Visualization", 
            variable=self.use_token_view
        )
        self.toggle_switch.pack(side=tk.LEFT, padx=10)
        
        self.btn_debug = ttk.Button(self.btn_frame, text="Show Context", command=self.show_context_debug)
        self.btn_debug.pack(side=tk.LEFT, padx=5)

        self.btn_clear = ttk.Button(self.btn_frame, text="Clear History", command=self.clear_history)
        self.btn_clear.pack(side=tk.LEFT, padx=5)
        
        self.context_bar = ttk.Progressbar(root, length=500, mode="determinate")
        self.context_bar.pack(fill="x", padx=10)
        
        self.token_label = tk.Label(root, text="Prompt Tokens: 0 | Response Tokens: 0", font=("Arial", 11))
        self.token_label.pack(pady=5)

        # --- Chat Box ---
        self.chat_box = tk.Text(
            root, 
            wrap="word", 
            font=("Arial", 12), 
            state="disabled", 
            height=15, 
            bg="#1e1e1e", 
            insertbackground="white"
        )
        self.chat_box.pack(fill="both", expand=True, padx=10, pady=10)

        # --- Input ---
        self.input_box = tk.Text(root, height=4, font=("Arial", 12), wrap="word")
        self.input_box.pack(fill="x", padx=10, pady=5)
        self.input_box.bind("<Return>", self.enter_pressed)

        self.send_button = tk.Button(root, text="Send", command=self.send_message)
        self.send_button.pack(pady=5)

        self.timer_label = tk.Label(root, text="Timer: 0.0s", font=("Arial", 12))
        self.timer_label.pack(pady=5)

        # Base tags
        self.chat_box.tag_configure("system", foreground="#00bcd4", background="black")
        self.chat_box.tag_configure("user", foreground="white", background="black")
        self.chat_box.tag_configure("ai", foreground="#4caf50", background="black") 
        self.chat_box.tag_configure("token", foreground="#2196f3", font=("Consolas", 10, "italic"))
        
        # Reasoning mode tags
        self.chat_box.tag_configure("thinking_header", foreground="#ff9800", font=("Arial", 11, "bold", "italic"))
        self.chat_box.tag_configure("thinking_content", foreground="#b0bec5", background="#263238", font=("Consolas", 11, "italic"))

        self.timer_running = False
        self.start_time = 0.0

    def show_context_debug(self):
        """Opens a new window to visualize the current chat history list."""
        debug_win = tk.Toplevel(self.root)
        debug_win.title("Current Context Window")
        text_area = tk.Text(debug_win, width=80, height=25, font=("Consolas", 10))
        text_area.pack(fill="both", expand=True, padx=5, pady=5)
        
        history_str = json.dumps(self.chat_history, indent=4)
        text_area.insert(tk.END, history_str)
        text_area.config(state="disabled")

    def clear_history(self):
        self.chat_history = []
        self.chat_box.configure(state="normal")
        self.chat_box.delete("1.0", tk.END)
        self.chat_box.configure(state="disabled")
        self.display_message("System", "Context cleared. Starting fresh.")

    def display_message(self, sender, message, tag=None):
        self.chat_box.configure(state="normal")
        self.chat_box.insert(tk.END, f"{sender}: {message}\n", tag or sender.lower())
        self.chat_box.configure(state="disabled")
        self.chat_box.see(tk.END)

    def update_timer(self):
        if self.timer_running:
            elapsed = time.time() - self.start_time
            self.timer_label.config(text=f"Timer: {elapsed:.2f}s")
            self.root.after(100, self.update_timer)

    def stop_timer(self):
        self.timer_running = False

    def send_message(self, event=None):
        message = self.input_box.get("1.0", tk.END).strip()
        if not message: return

        if self.use_token_view.get():
            tokens = self.tokenizer.tokenize(message)
            self.display_message("System", f"Tokenized View: {' | '.join(tokens)}", tag="token")

        self.chat_history.append({"role": "user", "content": message})
        self.display_message("You", message)
        
        self.input_box.delete("1.0", tk.END)
        self.timer_running = True
        self.start_time = time.time()
        self.update_timer()

        threading.Thread(target=self.send_to_ai, daemon=True).start()

    def send_to_ai(self):
        try:
            full_response = ""
            is_thinking = False
            header_printed = False
            buffer = ""

            self.root.after(0, self.prepare_ai_stream_ui)

            for event in self.client.chat(
                messages=self.chat_history,
                model="qwen3-vl:8b",
                think=True,
                context_size=self.context_size,
            ):
                content = event.content
                full_response += content
                buffer += content

                if "<think>" in buffer:
                    is_thinking = True
                    buffer = buffer.replace("<think>", "")
                    if not header_printed:
                        self.root.after(0, self.stream_to_ui, ">> AI Internal Thought Process:", "thinking_header")
                        header_printed = True

                if "</think>" in buffer:
                    is_thinking = False
                    buffer = buffer.replace("</think>", "")
                    self.root.after(0, self.stream_to_ui, ">> Final Response:", "thinking_header")

                if buffer:
                    self.root.after(0, self.stream_to_ui, buffer, "thinking_content" if is_thinking else "ai")
                    buffer = ""

                if event.done:
                    self.root.after(0, self.update_token_label, event.prompt_tokens, event.response_tokens)

            self.chat_history.append({"role":"assistant","content":full_response})
            self.root.after(0,self.stop_timer)

        except Exception as e:
            self.root.after(0,self.display_message,"AI",f"Error: {e}")
            self.root.after(0,self.stop_timer)

    def prepare_ai_stream_ui(self):
        self.chat_box.configure(state="normal")
        self.chat_box.insert(tk.END, "AI: ")
        self.chat_box.configure(state="disabled")

    def stream_to_ui(self, chunk, tag):
        self.chat_box.configure(state="normal")
        self.chat_box.insert(tk.END, chunk, tag)
        self.chat_box.configure(state="disabled")
        self.chat_box.see(tk.END)

    def update_token_label(self, prompt_tokens, response_tokens):
        used = (prompt_tokens / self.context_size) * 100
        self.context_bar["value"] = used
        self.token_label.config(
            text=f"Prompt Tokens: {prompt_tokens:,} | Response Tokens: {response_tokens:,} | Used: {used:.2f}%"
        )

    def enter_pressed(self, event):
        if not event.state & 0x1:
            self.send_message()
            return "break"

if __name__ == "__main__":
    root = tk.Tk()
    app = ChatApp(root)
    root.mainloop()