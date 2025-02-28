"use client";
import React, { Suspense, useState } from "react";
import { ForgotPassword, Login, SignUp } from "./components/Users";
import { useSearchParams } from "next/navigation";
import { useUserAuth } from "@/app/components/UserWrapper";

function Page() {
  const lang: string | null = useSearchParams().get("type") || null;

  const { isUserLoggedIn, handleLoggoutUser } = useUserAuth();
  console.log(isUserLoggedIn)
  if (isUserLoggedIn)
    return (
      <>
        User Logged In{" "}
        <button
          onClick={() => {
            handleLoggoutUser();
          }}
        >
          Loggout
        </button>
      </>
    );

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg overflow-hidden">
        <Form type={lang} />
      </div>
    </div>
  );
}

function Form({ type }) {
  const [toggleType, setToggleType] = useState(type);
  return (
    <>
      <div className="flex justify-around border-b">
        <div
          onClick={() => setToggleType("0")}
          className={`w-1/2 py-3 text-center hover:bg-gray-300 cursor-pointer ${
            toggleType === "0" ? "bg-gray-300" : ""
          }`}
        >
          Login
        </div>
        <div
          onClick={() => setToggleType("1")}
          className={`w-1/2 py-3 hover:bg-gray-300 text-center cursor-pointer ${
            toggleType === "1" ? "bg-gray-300" : ""
          }`}
        >
          Sign Up
        </div>
      </div>
      {toggleType != "2" && (
        <div className="w-full my-2 flex cursor-pointer items-center text-blue-400 justify-center">
          <span className=" underline " onClick={() => setToggleType("2")}>
            Forgot password ???
          </span>
        </div>
      )}
      <div className="p-6">
        <>{toggleType === "0" && <Login />}</>
        <>{toggleType === "1" && <SignUp />}</>
        <>{toggleType === "2" && <ForgotPassword />}</>
      </div>
    </>
  );
}

export default function page() {
  return (
    <>
      <Suspense>
        <Page />
      </Suspense>
    </>
  );
}