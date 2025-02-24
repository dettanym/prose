from collections.abc import Generator
from colorsys import hls_to_rgb, rgb_to_hls
from typing import Generic, TypeVar

import matplotlib.colors as mc

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


# copied from https://stackoverflow.com/a/49601444
def lighten_color(color, amount=0.5):
    """
    Lightens the given color by multiplying (1-luminosity) by the given amount.
    Input can be matplotlib color string, hex string, or RGB tuple.

    Examples:
    >> lighten_color('g', 0.3)
    >> lighten_color('#F034A3', 0.6)
    >> lighten_color((.3,.55,.1), 0.5)
    """

    try:
        c = mc.cnames[color]
    except:
        c = color

    h, l, s = rgb_to_hls(*mc.to_rgb(c))
    return hls_to_rgb(h, max(0, min(1, amount * l)), s)
