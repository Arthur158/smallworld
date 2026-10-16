import { roomNameAdjectives } from '../data/roomNameAdjectives';
import { roomNamePlaces } from '../data/roomNamePlaces';
import { createRandomRoomName } from './roomNames';

test('the room-name vocabulary has 400 distinct adjectives and 400 distinct places', () => {
  for (const words of [roomNameAdjectives, roomNamePlaces]) {
    expect(words).toHaveLength(400);
    expect(new Set(words).size).toBe(400);
    expect(words.every(word => word.trim() === word && word.length > 0)).toBe(true);
  }
});

test('each name independently rolls both parts, including the ends of each vocabulary', () => {
  const random = jest.spyOn(Math, 'random');
  try {
    random.mockReturnValueOnce(0).mockReturnValueOnce(0.999999);
    expect(createRandomRoomName()).toBe(`The ${roomNameAdjectives[0]} ${roomNamePlaces[399]}`);
    random.mockReturnValueOnce(0.999999).mockReturnValueOnce(0);
    expect(createRandomRoomName()).toBe(`The ${roomNameAdjectives[399]} ${roomNamePlaces[0]}`);
    expect(random).toHaveBeenCalledTimes(4);
  } finally {
    random.mockRestore();
  }
});
