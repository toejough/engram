// The SDK result message carries the running total cost of the session (a resumed session carries the earlier turns
// too), so a turn costs the change since the previous result.
export function turnCost(previousTotal, total) {
  if (typeof total !== 'number') return 0
  return total >= previousTotal ? total - previousTotal : total
}
