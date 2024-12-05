import { Polygon } from '../types/Board';

export function parseAreaFile(text: string): Polygon[] {
  // Regex to match <area> tags with coords and shape attributes
  const areaRegex = /<area[^>]*coords="([^"]+)"[^>]*shape="([^"]+)"[^>]*>/g;
  const polygons: Polygon[] = [];
  let match: RegExpExecArray | null;

  // Extract and parse each area tag
  while ((match = areaRegex.exec(text)) !== null) {
    const coords = match[1].split(',').map(Number); // Convert coords to numbers
    const shape = match[2]; // Extract the shape
    polygons.push({ coords });
  }

  return polygons;
}
