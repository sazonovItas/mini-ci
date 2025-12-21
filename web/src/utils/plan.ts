interface PlanNode {
  ref?: { id: string };
  next?: PlanNode;
}

/**
 * Traverses a linked-list plan and returns the data array sorted by execution order.
 */
export function sortByPlan<T extends { id: string }>(items: T[], plan: PlanNode | undefined): T[] {
  if (!plan || !items.length) return items;

  // 1. Extract ordered IDs from the plan
  const orderedIds: string[] = [];
  let current: PlanNode | undefined = plan;

  while (current) {
    if (current.ref?.id) {
      orderedIds.push(current.ref.id);
    }
    current = current.next;
  }

  // 2. Sort the items array to match the plan order
  // Items not in the plan (shouldn't happen) go to the end
  return [...items].sort((a, b) => {
    const idxA = orderedIds.indexOf(a.id);
    const idxB = orderedIds.indexOf(b.id);

    // Handle missing IDs gracefully
    const safeA = idxA === -1 ? 9999 : idxA;
    const safeB = idxB === -1 ? 9999 : idxB;

    return safeA - safeB;
  });
}
