export function range(start: number, stop: number, step = 1) {
  return Array.from(
    { length: (stop - start) / step + 1 },
    (_, index) => start + index * step,
  )
}

export function dropLeftWhile<T>(
  array: ReadonlyArray<T>,
  predicate: (item: T) => boolean,
): ReadonlyArray<T> {
  const i = array.findIndex((_) => !predicate(_))
  return i > -1 ? array.slice(i) : array
}

export function dropRightWhile<T>(
  array: ReadonlyArray<T>,
  predicate: (item: T) => boolean,
): ReadonlyArray<T> {
  const i = array.findLastIndex((_) => !predicate(_))
  return i > -1 ? array.slice(0, i + 1) : array
}

export function typeCheck(_: true): void {}

export function absurd<A>(_: never): A {
  throw new Error("Called `absurd` function which should be uncallable")
}
