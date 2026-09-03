"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { CikisYapDocument, OrganizasyonumDocument, UyelerDocument } from "@/generated/graphql";
import { HesapTipi, Rol } from "@/lib/enums";
import type { Member, Organization } from "@/lib/graphql-types";
import { AuthExpiredError, gqlRequest, isForbidden, restoreSession } from "./client";
import {
  clearSession,
  getAccessToken,
  isPublicPath,
  setAccessToken,
  setRefreshToken,
  userIdFromAccessToken,
} from "./session";

type AuthState = {
  ready: boolean;
  user: Member | null;
  org: Organization | null;
  userId: string | null;
  isViewer: boolean;
  isOwner: boolean;
  canWrite: boolean;
  canManageMembers: boolean;
  canViewMembers: boolean;
  setTokens: (access: string, refresh: string) => Promise<void>;
  refreshProfile: () => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState<Member | null>(null);
  const [org, setOrg] = useState<Organization | null>(null);
  const [userId, setUserId] = useState<string | null>(null);

  const refreshProfile = useCallback(async () => {
    const token = getAccessToken();
    if (!token) {
      setUser(null);
      setOrg(null);
      setUserId(null);
      return;
    }
    const uid = userIdFromAccessToken(token);
    setUserId(uid);
    try {
      const organization = await gqlRequest(OrganizasyonumDocument);
      const orgDoc = organization.organizasyonum ?? null;
      setOrg(orgDoc);
      try {
        const members = await gqlRequest(UyelerDocument);
        const me = members.uyeler.find((u) => u.id === uid) ?? null;
        setUser(me);
      } catch (err) {
        if (err instanceof AuthExpiredError) {
          throw err;
        }
        if (isForbidden(err) && uid) {
          setUser({
            id: uid,
            eposta: "",
            rol: Rol.Goruntuleyici,
            hesapTipi: orgDoc ? HesapTipi.Kurumsal : HesapTipi.Bireysel,
          });
          return;
        }
        throw err;
      }
    } catch (err) {
      if (err instanceof AuthExpiredError) {
        setUser(null);
        setOrg(null);
        setUserId(null);
        return;
      }
      throw err;
    }
  }, []);

  const setTokens = useCallback(
    async (access: string, refresh: string) => {
      setAccessToken(access);
      setRefreshToken(refresh);
      await refreshProfile();
    },
    [refreshProfile],
  );

  const logout = useCallback(async () => {
    try {
      if (getAccessToken()) {
        await gqlRequest(CikisYapDocument);
      }
    } catch {
      /* oturum zaten kapalı olabilir */
    }
    clearSession();
    setUser(null);
    setOrg(null);
    setUserId(null);
    router.replace("/giris/");
  }, [router]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await restoreSession();
        if (getAccessToken()) {
          await refreshProfile();
        }
      } catch {
        clearSession();
      } finally {
        if (!cancelled) {
          setReady(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [refreshProfile]);

  useEffect(() => {
    if (!ready) {
      return;
    }
    const path = pathname.replace(/\/$/, "") || "/";
    if (!getAccessToken() && !isPublicPath(path)) {
      router.replace("/giris/");
    }
  }, [ready, pathname, router]);

  const role = user?.rol;
  const value = useMemo<AuthState>(
    () => ({
      ready,
      user,
      org,
      userId,
      isViewer: role === Rol.Goruntuleyici,
      isOwner: role === Rol.Sahip || user?.hesapTipi === HesapTipi.Bireysel,
      canWrite: Boolean(user) && role !== Rol.Goruntuleyici,
      canManageMembers: role === Rol.Sahip,
      canViewMembers: Boolean(org) && (role === Rol.Sahip || role === Rol.Yonetici),
      setTokens,
      refreshProfile,
      logout,
    }),
    [ready, user, org, userId, role, setTokens, refreshProfile, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth AuthProvider dışında");
  }
  return ctx;
}
