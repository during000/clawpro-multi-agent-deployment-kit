import { createContext, useContext, useState, ReactNode } from "react";

interface UserRoleContextType {
  isAdmin: boolean;
  toggleRole: () => void;
}

const UserRoleContext = createContext<UserRoleContextType>({
  isAdmin: true,
  toggleRole: () => {},
});

export function UserRoleProvider({ children }: { children: ReactNode }) {
  // 默认为管理员，演示时可切换
  const [isAdmin, setIsAdmin] = useState(true);

  const toggleRole = () => setIsAdmin((prev) => !prev);

  return (
    <UserRoleContext.Provider value={{ isAdmin, toggleRole }}>
      {children}
    </UserRoleContext.Provider>
  );
}

export function useUserRole() {
  return useContext(UserRoleContext);
}
