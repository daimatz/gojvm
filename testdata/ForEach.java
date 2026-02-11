public class ForEach {
    public static void main(String[] args) {
        // for-each on array
        int[] nums = {10, 20, 30};
        int sum = 0;
        for (int n : nums) {
            sum += n;
        }
        System.out.println(sum);  // 60

        // for-each on String array
        String[] words = {"Hello", "World"};
        for (String w : words) {
            System.out.println(w);
        }
    }
}
