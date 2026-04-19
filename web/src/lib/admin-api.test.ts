import { formatCompactNumber } from "./admin-api";

function assertEqual(actual: unknown, expected: unknown, message: string) {
  if (actual !== expected) {
    throw new Error(`${message}: expected ${expected}, got ${actual}`);
  }
}

assertEqual(formatCompactNumber(999), "999", "should keep small numbers unscaled");
assertEqual(formatCompactNumber(1_200), "1.2k", "should render thousands with k suffix");
assertEqual(formatCompactNumber(1_000_000), "1m", "should render millions with m suffix");
assertEqual(formatCompactNumber(1_550_000_000), "1.6b", "should render billions with b suffix");
assertEqual(formatCompactNumber(-12_300), "-12.3k", "should preserve sign when compacting");

console.log("formatCompactNumber tests passed");
