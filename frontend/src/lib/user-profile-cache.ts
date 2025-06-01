interface UserProfile {
  name: string;
}

interface CachedUserProfile {
  profile: UserProfile;
  expiresAt: number;
}

export class UserProfileCache {
  private cache = new Map<string, CachedUserProfile>();
  private readonly cacheExpiry = 10 * 60 * 1000; // 10 minutes in milliseconds

  constructor() {
    // Clean up expired entries every minute
    setInterval(() => {
      this.cleanupExpired();
    }, 60 * 1000);
  }

  async getUserProfile(userID: string): Promise<UserProfile> {
    // Check cache first
    const cached = this.cache.get(userID);
    if (cached && Date.now() < cached.expiresAt) {
      return cached.profile;
    }

    // Fetch from API
    try {
      const response = await fetch(`/api/user/${userID}/profile`, {
        credentials: 'include',
      });

      if (!response.ok) {
        // Fallback to userID as name if API fails
        const fallbackProfile: UserProfile = { name: userID };
        
        // Cache the fallback profile with shorter expiry (1 minute)
        this.cache.set(userID, {
          profile: fallbackProfile,
          expiresAt: Date.now() + 60 * 1000
        });
        
        return fallbackProfile;
      }

      const profile: UserProfile = await response.json();
      
      // Cache the successful result
      this.cache.set(userID, {
        profile,
        expiresAt: Date.now() + this.cacheExpiry
      });

      return profile;
    } catch (error) {
      console.error('Failed to fetch user profile:', error);
      
      // Fallback to userID as name
      const fallbackProfile: UserProfile = { name: userID };
      
      // Cache the fallback profile with shorter expiry (1 minute)
      this.cache.set(userID, {
        profile: fallbackProfile,
        expiresAt: Date.now() + 60 * 1000
      });
      
      return fallbackProfile;
    }
  }

  private cleanupExpired(): void {
    const now = Date.now();
    for (const [userID, cached] of this.cache.entries()) {
      if (now >= cached.expiresAt) {
        this.cache.delete(userID);
      }
    }
  }

  // Clear all cached entries
  clear(): void {
    this.cache.clear();
  }

  // Get cache stats for debugging
  getStats(): { size: number; entries: string[] } {
    return {
      size: this.cache.size,
      entries: Array.from(this.cache.keys())
    };
  }
}

// Global instance
export const userProfileCache = new UserProfileCache(); 