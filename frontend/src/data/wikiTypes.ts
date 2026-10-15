export interface WikiArticle {
  id: string;
  kind: 'race' | 'trait' | 'power' | 'piece' | 'terrain';
  name: string;
  summary: string;
  details: string[];
  pieces: string[];
  sources: { label: string; url: string }[];
  note?: string;
}
