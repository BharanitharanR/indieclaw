from dataclasses import dataclass
from typing import Optional


@dataclass
class ChatEvent:
    """
    Represents one streamed event from the Rust Gateway.
    """

    content: str = ""
    done: bool = False

    prompt_tokens: int = 0
    response_tokens: int = 0

    raw: Optional[dict] = None