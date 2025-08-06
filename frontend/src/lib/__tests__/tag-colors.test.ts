import {
  colorClassToName,
  colorNameToClass,
  TAG_COLOR_MAP,
  TAG_CLASS_MAP,
  DEFAULT_TAG_COLOR,
  DEFAULT_TAG_CLASS,
  isValidColorName,
  isValidColorClass,
  getAvailableColorNames,
  getAvailableColorClasses,
} from '../tag-colors';

describe('tag-colors utilities', () => {
  describe('colorClassToName', () => {
    it('should convert valid color class to name', () => {
      expect(colorClassToName('bg-red-100 text-red-800')).toBe('red');
      expect(colorClassToName('bg-blue-100 text-blue-800')).toBe('blue');
    });

    it('should return default for invalid color class', () => {
      expect(colorClassToName('invalid-class')).toBe(DEFAULT_TAG_COLOR);
    });
  });

  describe('colorNameToClass', () => {
    it('should convert valid color name to class', () => {
      expect(colorNameToClass('red')).toBe('bg-red-100 text-red-800');
      expect(colorNameToClass('blue')).toBe('bg-blue-100 text-blue-800');
    });

    it('should return default for invalid color name', () => {
      expect(colorNameToClass('invalid-color')).toBe(DEFAULT_TAG_CLASS);
    });
  });

  describe('validation functions', () => {
    it('should validate color names correctly', () => {
      expect(isValidColorName('red')).toBe(true);
      expect(isValidColorName('invalid')).toBe(false);
    });

    it('should validate color classes correctly', () => {
      expect(isValidColorClass('bg-red-100 text-red-800')).toBe(true);
      expect(isValidColorClass('invalid-class')).toBe(false);
    });
  });

  describe('getAvailable functions', () => {
    it('should return all available color names', () => {
      const names = getAvailableColorNames();
      expect(names).toContain('red');
      expect(names).toContain('blue');
      expect(names).toHaveLength(Object.keys(TAG_COLOR_MAP).length);
    });

    it('should return all available color classes', () => {
      const classes = getAvailableColorClasses();
      expect(classes).toContain('bg-red-100 text-red-800');
      expect(classes).toContain('bg-blue-100 text-blue-800');
      expect(classes).toHaveLength(Object.values(TAG_COLOR_MAP).length);
    });
  });

  describe('mapping consistency', () => {
    it('should have consistent forward and reverse mappings', () => {
      Object.entries(TAG_COLOR_MAP).forEach(([name, className]) => {
        expect(TAG_CLASS_MAP[className]).toBe(name);
        expect(colorNameToClass(name)).toBe(className);
        expect(colorClassToName(className)).toBe(name);
      });
    });

    it('should have correct default values', () => {
      expect(DEFAULT_TAG_COLOR).toBe('gray');
      expect(DEFAULT_TAG_CLASS).toBe('bg-gray-100 text-gray-800');
      expect(TAG_COLOR_MAP[DEFAULT_TAG_COLOR]).toBe(DEFAULT_TAG_CLASS);
    });
  });
});