import { Polygon } from '../types/Board';

export function parseAreaFile(text: string): Polygon[] {
  // Regex to match <area> tags with coords and stack attributes
  const areaRegex = /<area[^>]*coords="([^"]+)"[^>]*stack="([^,]+),([^"]+)"[^>]*>/g;
  const polygons: Polygon[] = [];
  let match: RegExpExecArray | null;

  // Extract and parse each area tag
  while ((match = areaRegex.exec(text)) !== null) {
    const coords = match[1].split(',').map(Number); // Convert coords to numbers
    const stackX = Number(match[2]); // Extract and convert stackX
    const stackY = Number(match[3]); // Extract and convert stackY
    polygons.push({ coords, stackX, stackY });
  }

  return polygons;
}
