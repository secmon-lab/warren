import { useState, useEffect } from "react";
import { userProfileCache } from "@/lib/user-profile-cache";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { ANONYMOUS_USER_ID, ANONYMOUS_USER_NAME } from "@/constants/auth";
import { UserIcon } from "lucide-react";

interface UserNameProps {
  userID: string;
  className?: string;
  fallback?: string;
}

export function UserName({ userID, className = "", fallback }: UserNameProps) {
  const [name, setName] = useState<string>(fallback || userID);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;

    const fetchUserName = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // 匿名ユーザーの場合は固定の名前を使用
        if (userID === ANONYMOUS_USER_ID) {
          if (mounted) {
            setName(ANONYMOUS_USER_NAME);
          }
          return;
        }

        const profile = await userProfileCache.getUserProfile(userID);

        if (mounted) {
          setName(profile.name || fallback || userID);
        }
      } catch (err) {
        console.error("Failed to fetch user name:", err);
        if (mounted) {
          setError("Failed to load user name");
          setName(fallback || userID);
        }
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    };

    fetchUserName();

    return () => {
      mounted = false;
    };
  }, [userID, fallback]);

  if (isLoading) {
    return (
      <span
        className={`animate-pulse bg-muted rounded text-transparent ${className}`}>
        {fallback || userID}
      </span>
    );
  }

  if (error) {
    return (
      <span className={`text-muted-foreground ${className}`} title={error}>
        {fallback || userID}
      </span>
    );
  }

  return (
    <span className={className} title={`User: ${userID}`}>
      {name}
    </span>
  );
}

// Hook version for more advanced usage
export function useUserName(userID: string): {
  name: string;
  isLoading: boolean;
  error: string | null;
} {
  const [name, setName] = useState<string>(userID);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;

    const fetchUserName = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // 匿名ユーザーの場合は固定の名前を使用
        if (userID === ANONYMOUS_USER_ID) {
          if (mounted) {
            setName(ANONYMOUS_USER_NAME);
          }
          return;
        }

        const profile = await userProfileCache.getUserProfile(userID);

        if (mounted) {
          setName(profile.name || userID);
        }
      } catch (err) {
        console.error("Failed to fetch user name:", err);
        if (mounted) {
          setError("Failed to load user name");
          setName(userID);
        }
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    };

    fetchUserName();

    return () => {
      mounted = false;
    };
  }, [userID]);

  return { name, isLoading, error };
}

// Component that shows both avatar and name
interface UserWithAvatarProps {
  userID: string;
  className?: string;
  fallback?: string;
  avatarSize?: "sm" | "md" | "lg";
  showAvatar?: boolean;
}

export function UserWithAvatar({
  userID,
  className = "",
  fallback,
  avatarSize = "sm",
  showAvatar = true,
}: UserWithAvatarProps) {
  const { name, isLoading, error } = useUserName(userID);

  const avatarSizeClasses = {
    sm: "h-3 w-3",
    md: "h-5 w-5",
    lg: "h-8 w-8",
  };

  const displayName = name || fallback || "Unknown User";

  if (isLoading) {
    return (
      <div className={`flex items-center gap-1 ${className}`}>
        {showAvatar && (
          <div
            className={`${avatarSizeClasses[avatarSize]} bg-muted rounded-full animate-pulse`}
          />
        )}
        <span className="animate-pulse bg-muted rounded text-transparent">
          {fallback || "Loading..."}
        </span>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center gap-1 ${className}`} title={error}>
        {showAvatar && (
          <Avatar className={avatarSizeClasses[avatarSize]}>
            {userID !== ANONYMOUS_USER_ID ? (
              <>
                <AvatarImage src={`/api/user/${userID}/icon`} alt={displayName} />
                <AvatarFallback className="text-xs leading-none">
                  {displayName.charAt(0).toUpperCase()}
                </AvatarFallback>
              </>
            ) : (
              <AvatarFallback className="text-xs leading-none">
                <UserIcon className="h-3 w-3" />
              </AvatarFallback>
            )}
          </Avatar>
        )}
        <span className="text-muted-foreground">{displayName}</span>
      </div>
    );
  }

  return (
    <div className={`flex items-center gap-1 ${className}`}>
      {showAvatar && (
        <Avatar className={avatarSizeClasses[avatarSize]}>
          {userID !== ANONYMOUS_USER_ID ? (
            <>
              <AvatarImage src={`/api/user/${userID}/icon`} alt={displayName} />
              <AvatarFallback className="text-xs leading-none">
                {displayName.charAt(0).toUpperCase()}
              </AvatarFallback>
            </>
          ) : (
            <AvatarFallback className="text-xs leading-none">
              <UserIcon className="h-3 w-3" />
            </AvatarFallback>
          )}
        </Avatar>
      )}
      <span>{displayName}</span>
    </div>
  );
}
