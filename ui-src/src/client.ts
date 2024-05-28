import { CatalogObject } from "./types";

export async function getObjects(): Promise<CatalogObject[]> {
  const url = "https://api.restful-api.dev/objects";
  try {
    const objects = await fetch(url);
    return objects.json();
  } catch (e) {
    throw new Error(`${e.message} ${url}`);
  }
}
