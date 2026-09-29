// Each gate test declares the contract operations it exercises; coverage.spec
// reads these declarations to prove all 33 are covered.
export function operations(
  ...ids: string[]
): { type: 'operation'; description: string }[] {
  return ids.map((description) => ({ type: 'operation', description }));
}
