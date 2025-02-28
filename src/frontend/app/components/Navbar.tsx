"use client";
import { axiosFetchAuth } from "@/lib/axiosConfig";
import Link from "next/link";
import React, { useEffect, useState } from "react";
import { UserAuthWrapper, useUserAuth } from "./UserWrapper";

function Navbar() {
  const [toggleNav, setToggleNav] = useState(false);
  const {isUserLoggedIn} = useUserAuth()
  const user = isUserLoggedIn




  return (
    <nav className="bg-gray-800 mob:p-4 p-0 w-full">
      <div className="container mx-auto flex justify-between items-center">
        <span className="text-white text-lg font-bold w-[11rem]">
          My Full Project
        </span>
        <div
          className={
            "flex gap-2 bg-gray-800 w-full absolute flex-col p-2 mob:p-0 mob:gap-0 mob:flex-row mob:justify-between mob:static mob:top-auto" +
            (toggleNav
              ? " top-10 transition-top"
              : " top-[-11rem] transition-top")
          }
        >
          <ul className="flex mob:space-x-4 mob:gap-0 gap-2 flex-col items-center justify-center mob:flex-row ">
            <li className="text-white">Products</li>
            <li className="text-white">Solutions</li>
          </ul>
          <ul className="flex mob:space-x-4 mob:gap-0 gap-2 items-center justify-center flex-col mob:flex-row ">
            <li className="text-white">Contact</li>
            <li onClick={() => setToggleNav(!toggleNav)} className="text-white">
              {!user ? (
                <Link href="/user/auth?type=0">Login</Link>
              ) : (
                <>Profile redirect pending</>
              )}
            </li>
            <li onClick={() => setToggleNav(!toggleNav)} className="text-white">
              {!user ? (
                <Link href="/user/auth?type=1" className="hover:underline">
                  Signup{" "}
                </Link>
              ) : (
                <button
                  onClick={() => {
                    localStorage.clear();
                    localStorage.removeItem("token");
                    window.location.replace("/user/auth?type=0");
                  }}
                  className="p-2 bg-red-600 rounded-md hover:bg-red-400 focus:outline-none focus:ring focus:border-blue-300"
                >
                  Logout
                </button>
              )}
            </li>
          </ul>
        </div>
        <span
          onClick={() => {
            setToggleNav(!toggleNav);
          }}
          className="text-white flex mob:hidden items-center text-4xl justify-center cursor-pointer"
        >
          &#8801;
        </span>
      </div>
    </nav>
  );
}

export default Navbar;
