export function useBlogRoutes() {
  function categoryPath(name: string) {
    return `/categories/${encodeURIComponent(name)}`
  }

  function tagPath(name: string) {
    return `/tags/${encodeURIComponent(name)}`
  }

  return {
    categoryPath,
    tagPath
  }
}
