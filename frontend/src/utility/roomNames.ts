import { roomNameAdjectives } from '../data/roomNameAdjectives';
import { roomNamePlaces } from '../data/roomNamePlaces';

export function createRandomRoomName(): string {
  const adjective = roomNameAdjectives[Math.floor(Math.random() * roomNameAdjectives.length)];
  const place = roomNamePlaces[Math.floor(Math.random() * roomNamePlaces.length)];
  return `The ${adjective} ${place}`;
}
