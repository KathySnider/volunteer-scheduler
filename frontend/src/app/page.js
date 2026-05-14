"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { isAuthenticated } from "./lib/api";

export default function HomePage() {
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace("/events");
    } else {
      router.replace("/login");
    }
  }, [router]);

  return null;
}
