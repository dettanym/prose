from collections.abc import Generator
from typing import Generic, TypeVar

T = TypeVar("T")
U = TypeVar("U")
V = TypeVar("V")


class ValuedGenerator(Generic[T, U, V]):
    value: V

    def __init__(self, generator: Generator[T, U, V]):
        self.generator = generator
        super().__init__()

    def __iter__(self) -> Generator[T, U, V]:
        self.value = yield from self.generator
        return self.value
