export function codeBlockId(language: string, code: string) {
  let hash = 2166136261;
  for (const char of `${language}\0${code}`) {
    hash ^= char.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}
