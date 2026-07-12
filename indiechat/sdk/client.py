import json
import requests

from .models import ChatEvent
from .exceptions import (
    GatewayConnectionError,
    GatewayResponseError,
)


class RustAIClient:

    def __init__(
        self,
        base_url="http://localhost:8080",
        timeout=600,
    ):
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    ####################################################################
    # Chat
    ####################################################################
    def chat(
        self,
        messages,
        model="qwen3:8b",
        think=True,
        context_size=40960,
        temperature=None,
        extra_options=None,
    ):

        options = {
            "num_ctx": context_size,
            "think": think,
        }

        if temperature is not None:
            options["temperature"] = temperature

        if extra_options:
            options.update(extra_options)

        payload = {
            "model": model,
            "messages": messages,
            "stream": False,
            "options": options,
        }

        url = f"{self.base_url}/api/v1/chat"

        try:

            response = requests.post(
                url,
                json=payload,
                timeout=self.timeout,
            )

        except requests.exceptions.RequestException as e:
            raise GatewayConnectionError(str(e))

        if response.status_code != 200:
            raise GatewayResponseError(
                f"Gateway returned {response.status_code}\n"
                f"{response.text}"
            )

        data = response.json()

        message = data.get("message", {})
        content = message.get("content", "")

        yield ChatEvent(
            content=content,
            done=True,
            prompt_tokens=data.get("prompt_eval_count", 0),
            response_tokens=data.get("eval_count", 0),
            raw=data,
        )
    ####################################################################
    # Generate API
    ####################################################################

    def generate(
        self,
        prompt,
        model="qwen3:8b",
        context_size=40960,
        think=True,
    ):

        payload = {
            "model": model,
            "prompt": prompt,
            "stream": True,
            "options": {
                "num_ctx": context_size,
                "think": think,
            },
        }

        url = f"{self.base_url}/api/v1/generate"

        try:

            response = requests.post(
                url,
                json=payload,
                stream=True,
                timeout=self.timeout,
            )

        except requests.exceptions.RequestException as e:
            raise GatewayConnectionError(str(e))

        if response.status_code != 200:
            raise GatewayResponseError(
                f"Gateway returned {response.status_code}\n"
                f"{response.text}"
            )

        for line in response.iter_lines():

            if not line:
                continue

            data = json.loads(line.decode())

            yield ChatEvent(
                content=data.get("response", ""),
                done=data.get("done", False),
                prompt_tokens=data.get(
                    "prompt_eval_count",
                    0,
                ),
                response_tokens=data.get(
                    "eval_count",
                    0,
                ),
                raw=data,
            )

    ####################################################################
    # Health Check
    ####################################################################

    def ping(self):

        try:

            r = requests.get(
                f"{self.base_url}/health",
                timeout=5,
            )

            return r.status_code == 200

        except Exception:
            return False