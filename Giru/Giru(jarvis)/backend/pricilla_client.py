from __future__ import annotations

import time

import requests


class PricillaClient:
    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def _get(self, path: str) -> dict | list | None:
        last_error: Exception | None = None
        for attempt in range(3):
            try:
                response = requests.get(f"{self.base_url}{path}", timeout=self.timeout)
                response.raise_for_status()
                return response.json()
            except Exception as exc:
                last_error = exc
                time.sleep(0.4 * (attempt + 1))
        if last_error:
            raise last_error
        return None

    def get_status(self) -> dict | None:
        return self._get("/api/v1/status")

    def get_missions(self) -> list:
        data = self._get("/api/v1/missions")
        return data if isinstance(data, list) else []

    def get_mission(self, mission_id: str) -> dict | None:
        return self._get(f"/api/v1/missions/{mission_id}")

    def get_targeting_metrics(self) -> dict | None:
        data = self._get("/api/v1/metrics/targeting")
        return data if isinstance(data, dict) else None

    def get_hit_probability(self, mission_id: str) -> dict | None:
        return self._get(f"/api/v1/guidance/probability/{mission_id}")
