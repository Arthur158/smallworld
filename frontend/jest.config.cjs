module.exports = {
  rootDir: './src', // Update this path if necessary
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  moduleNameMapper: {
    '^.+\\.module\\.(css|sass|scss)$': 'identity-obj-proxy',
    '^.+\\.(css|sass|scss)$': '<rootDir>/../__mocks__/styleMock.js',
    '^.+\\.(jpg|jpeg|png|gif|webp|svg)$': '<rootDir>/../__mocks__/fileMock.js',
    '^leaflet$': '<rootDir>/../__mocks__/leafletMock.js',
  },
  setupFilesAfterEnv: ['<rootDir>/../jest.setup.ts'],
  transformIgnorePatterns: [
    '/node_modules/(?!leaflet)', // Ignore all node_modules except leaflet
  ],
  transform: {
    '^.+\\.(ts|tsx)$': 'ts-jest', // Use ts-jest for TypeScript
    '^.+\\.(js|jsx)$': 'babel-jest',
  },
};
