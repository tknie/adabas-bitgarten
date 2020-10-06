import Vue from 'vue';

export const config = {
   Url,
};

export function Url() {
   // console.log('Mode:' + process.env.NODE_ENV);
   console.log('URLX: ' + JSON.stringify(window.location));
   if (process.env.NODE_ENV === 'development') {
      //  return 'http://vangogh.fritz.box:8130';
      return 'http://localhost:8130';
      // return 'http://tiger:8130';
   }
   return window.location.origin;
}

export function AppName() {
   if (process.env.NODE_ENV === 'development') {
      return '/';
   }
   return '/app';
}
