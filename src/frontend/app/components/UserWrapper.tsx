"use client";
import React, {
  createContext,
  useState,
  useContext,
  useEffect,
  ReactNode,
} from "react";
import { axiosFetchAuth } from "@/lib/axiosConfig";

type AuthContextType = {
  isUserLoggedIn: boolean;
  wsConnection: WebSocket | null;
  setIsUserLoggedIn: React.Dispatch<React.SetStateAction<boolean>>;
  handleLoggoutUser: () => void;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

type UserAuthWrapperProps = {
  children: ReactNode;
};

export function UserAuthWrapper({ children }: UserAuthWrapperProps) {
  const [isUserLoggedIn, setIsUserLoggedIn] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [ws, setWS] = useState<WebSocket | null>(null);

  useEffect(() => {
    const token: string | null = localStorage.getItem("token");

    if (!token) {
      setIsLoading(false);
      setIsUserLoggedIn(false);
      return;
    }

    const checkAuthentication = async () => {
      try {
        const response = await axiosFetchAuth().get(`/isLoggedIn`, {
          headers: {
            Authorization: token,
          },
        });
        if (response.status === 200) {
          setIsUserLoggedIn(true);
        }
      } catch (error) {
        console.error("Authentication check failed:", error);
        setIsUserLoggedIn(false);
      } finally {
        setIsLoading(false);
      }
    };

    checkAuthentication();
  }, []);

  useEffect(() => {
    if (isUserLoggedIn && ws === null) {
      const token: string | null = localStorage.getItem("token");
      if (token) {
        console.log("hit ws");
        const websocket: WebSocket = new WebSocket(
          `ws://api.iamanshik.online/ws?token=${token}`
        );
        setWS(websocket);
      }
    }
  }, [isUserLoggedIn, ws]);

  const handleLoggoutUser = (): void => {
    localStorage.removeItem("token");
    setIsUserLoggedIn(false);
    window.location.reload();
  };

  const value: AuthContextType = {
    isUserLoggedIn,
    setIsUserLoggedIn,
    handleLoggoutUser,
    wsConnection: ws,
  };

  if (isLoading) return <>Loading User page...</>;
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useUserAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useUserAuth must be used within a UserAuthWrapper");
  }
  return context;
}
