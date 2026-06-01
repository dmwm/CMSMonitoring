from datetime import datetime, timedelta
from typing import Optional


def generate_date_ranges(
    start: Optional[datetime], end: Optional[datetime]
) -> list[tuple[datetime, datetime]]:
    if start > end:
        raise ValueError(f"Start date: {start} cannot be later than end date: {end}.")
    ranges = []
    current_start = start
    while current_start <= end:
        current_end = min(current_start + timedelta(days=30), end)
        ranges.append((current_start, current_end))
        current_start = current_end + timedelta(days=1)
    return ranges
