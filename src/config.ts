import Vue from 'vue';

export const config = {
    Url,
};

export function Url() {
   // console.log('Mode:' + process.env.NODE_ENV);
   if (process.env.NODE_ENV === 'development') {
    //  return 'http://vangogh.fritz.box:8130';
      return 'http://localhost:8130';
//      return 'http://tnas:8140';
   }
   return '.';
}
